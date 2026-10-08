# Sentiment inference service

This service loads the trained Trading Wens FinBERT checkpoint once and exposes calibrated batch sentiment inference to the Go API.

The model files are deployment artifacts and are intentionally not committed. Set `NLP_MODEL_DIR` to the extracted sentiment checkpoint directory containing `model.safetensors`, tokenizer files, `labels.json`, and `calibration.json`.

```powershell
python -m venv .venv
.\.venv\Scripts\Activate.ps1
python -m pip install -r ml\requirements.txt
$env:NLP_MODEL_DIR = 'C:\path\to\Trading-Wens-NLP-Training\trading_wens_ml\models\sentiment\best'
$env:NLP_MODEL_VERSION = 'sentiment-2026-10-08'
python ml\inference\server.py
```

`GET /health` reports the loaded model identity. `POST /v1/sentiment` accepts a JSON object with a non-empty `texts` array and returns one prediction per input in the same order.
