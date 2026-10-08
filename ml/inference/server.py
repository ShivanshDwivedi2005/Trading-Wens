from __future__ import annotations

import json
import math
import os
import threading
from dataclasses import dataclass
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any

import numpy as np
import torch
from transformers import BertForSequenceClassification, BertTokenizerFast


@dataclass(frozen=True)
class Settings:
    host: str
    port: int
    model_dir: Path
    model_name: str
    model_version: str
    max_batch_size: int

    @classmethod
    def from_environment(cls) -> "Settings":
        model_dir_value = os.environ.get("NLP_MODEL_DIR", "").strip()
        if not model_dir_value:
            raise RuntimeError("NLP_MODEL_DIR is required")
        host, separator, port = os.environ.get("NLP_LISTEN_ADDRESS", "127.0.0.1:8090").rpartition(":")
        if not separator or not host:
            raise RuntimeError("NLP_LISTEN_ADDRESS must use host:port")
        return cls(
            host=host,
            port=int(port),
            model_dir=Path(model_dir_value).expanduser().resolve(),
            model_name=os.environ.get("NLP_MODEL_NAME", "").strip()
            or "trading-wens-sentiment",
            model_version=os.environ.get("NLP_MODEL_VERSION", "").strip() or "local",
            max_batch_size=int(os.environ.get("NLP_MAX_BATCH_SIZE", "64")),
        )


class SentimentModel:
    def __init__(self, settings: Settings) -> None:
        if not settings.model_dir.is_dir():
            raise RuntimeError(f"NLP model directory does not exist: {settings.model_dir}")
        self.settings = settings
        self.tokenizer = BertTokenizerFast.from_pretrained(settings.model_dir, local_files_only=True)
        self.model = BertForSequenceClassification.from_pretrained(
            settings.model_dir, local_files_only=True
        ).eval()
        self.temperature = float(
            json.loads((settings.model_dir / "calibration.json").read_text(encoding="utf-8"))[
                "temperature"
            ]
        )
        self.labels = json.loads(
            (settings.model_dir / "labels.json").read_text(encoding="utf-8")
        )
        self.inference_lock = threading.Lock()

    @torch.inference_mode()
    def analyze(self, texts: list[str]) -> list[dict[str, Any]]:
        encoded = self.tokenizer(
            texts,
            return_tensors="pt",
            padding=True,
            truncation=True,
            max_length=192,
        )
        with self.inference_lock:
            logits = self.model(**encoded).logits.cpu().numpy()
        predictions: list[dict[str, Any]] = []
        for row in logits:
            scaled = row / self.temperature
            scaled -= scaled.max()
            probabilities = np.exp(scaled)
            probabilities /= probabilities.sum()
            winner = int(probabilities.argmax())
            entropy = -float(np.sum(probabilities * np.log(np.clip(probabilities, 1e-12, 1.0))))
            predictions.append(
                {
                    "label": self.labels[str(winner)],
                    "confidence": round(float(probabilities[winner]), 6),
                    "sentiment_score": round(float(probabilities[2] - probabilities[0]), 6),
                    "uncertainty": round(entropy / math.log(len(probabilities)), 6),
                    "probabilities": {
                        self.labels[str(index)]: round(float(probabilities[index]), 6)
                        for index in range(len(probabilities))
                    },
                }
            )
        return predictions


def create_handler(model: SentimentModel) -> type[BaseHTTPRequestHandler]:
    class Handler(BaseHTTPRequestHandler):
        def do_GET(self) -> None:
            if self.path != "/health":
                self._write_json(HTTPStatus.NOT_FOUND, {"error": "not_found"})
                return
            self._write_json(
                HTTPStatus.OK,
                {
                    "status": "ok",
                    "model": {
                        "name": model.settings.model_name,
                        "version": model.settings.model_version,
                    },
                },
            )

        def do_POST(self) -> None:
            if self.path != "/v1/sentiment":
                self._write_json(HTTPStatus.NOT_FOUND, {"error": "not_found"})
                return
            try:
                content_length = int(self.headers.get("Content-Length", "0"))
                if content_length <= 0 or content_length > 1_000_000:
                    raise ValueError("request body must be between 1 byte and 1 MB")
                payload = json.loads(self.rfile.read(content_length))
                texts = payload.get("texts")
                if not isinstance(texts, list) or not texts:
                    raise ValueError("texts must be a non-empty array")
                if len(texts) > model.settings.max_batch_size:
                    raise ValueError(
                        f"batch exceeds NLP_MAX_BATCH_SIZE={model.settings.max_batch_size}"
                    )
                if any(not isinstance(text, str) or not text.strip() for text in texts):
                    raise ValueError("every text must be a non-empty string")
                predictions = model.analyze([text.strip() for text in texts])
            except (ValueError, TypeError, json.JSONDecodeError) as error:
                self._write_json(HTTPStatus.BAD_REQUEST, {"error": str(error)})
                return
            self._write_json(
                HTTPStatus.OK,
                {
                    "model": {
                        "name": model.settings.model_name,
                        "version": model.settings.model_version,
                    },
                    "predictions": predictions,
                },
            )

        def log_message(self, message: str, *args: Any) -> None:
            print(f"nlp {self.address_string()} {message % args}")

        def _write_json(self, status: HTTPStatus, payload: dict[str, Any]) -> None:
            body = json.dumps(payload, separators=(",", ":")).encode("utf-8")
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

    return Handler


def main() -> None:
    settings = Settings.from_environment()
    model = SentimentModel(settings)
    server = ThreadingHTTPServer((settings.host, settings.port), create_handler(model))
    print(
        f"nlp listening on {settings.host}:{settings.port} "
        f"with {settings.model_name}@{settings.model_version}"
    )
    server.serve_forever()


if __name__ == "__main__":
    main()
