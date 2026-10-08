import type { NewsArticle } from "./news-api";
import type { PaperFill, TradingPosition } from "./trading-api";

export type PositionOutlook = {
  direction: "profit" | "loss" | "balanced";
  possibility: number;
  score: number;
  confidence: number;
  reasons: string[];
  sources: string[];
};

export function positionOutlook(
  position: TradingPosition | undefined,
  articles: NewsArticle[],
): PositionOutlook {
  if (!position) {
    return {
      direction: "balanced",
      possibility: 50,
      score: 5,
      confidence: 25,
      reasons: ["No open position is available for mark-to-market evidence."],
      sources: ["Alpaca paper account"],
    };
  }

  const analyzed = articles.flatMap((article) =>
    article.sentiment && article.matched_symbols.includes(position.symbol)
      ? [article.sentiment]
      : [],
  );
  const sentiment = analyzed.length
    ? weightedAverage(
        analyzed.map((item) => item.score),
        analyzed.map((item) => item.confidence),
      )
    : 0;
  const positionSignal = clamp(position.unrealized_pl_percent / 12, -1, 1);
  const dailySignal = clamp(position.change_today / 6, -1, 1);
  const composite = positionSignal * 0.4 + dailySignal * 0.25 + sentiment * 0.35;
  const direction = composite > 0.08 ? "profit" : composite < -0.08 ? "loss" : "balanced";
  const possibility = Math.round(50 + Math.min(0.4, Math.abs(composite)) * 100);
  const averageModelConfidence = analyzed.length
    ? analyzed.reduce((sum, item) => sum + item.confidence, 0) / analyzed.length
    : 0;
  const confidence = Math.round(
    clamp(0.42 + Math.min(analyzed.length, 5) * 0.045 + averageModelConfidence * 0.26, 0, 0.9) *
      100,
  );
  const signedPL = `${position.unrealized_pl >= 0 ? "+" : ""}${position.unrealized_pl.toFixed(2)}`;
  const reasons = [
    `Alpaca reports ${signedPL} USD unrealized P/L (${position.unrealized_pl_percent >= 0 ? "+" : ""}${position.unrealized_pl_percent.toFixed(2)}%).`,
    `Today's provider move is ${position.change_today >= 0 ? "+" : ""}${position.change_today.toFixed(2)}%.`,
    analyzed.length
      ? `${analyzed.length} model-scored headline${analyzed.length === 1 ? "" : "s"} contribute a confidence-weighted sentiment of ${sentiment >= 0 ? "+" : ""}${sentiment.toFixed(2)}.`
      : "No current model-scored headline matched this symbol, so news contributes a neutral input.",
  ];
  const sources = ["Alpaca paper position and mark prices"];
  if (analyzed.length) sources.push("Trading Wens FinBERT sentiment on Alpaca and GDELT news");

  return {
    direction,
    possibility,
    score: Number((possibility / 10).toFixed(1)),
    confidence,
    reasons,
    sources,
  };
}

export function fillMarkDelta(fill: PaperFill, currentPrice: number | undefined) {
  if (!currentPrice || currentPrice <= 0) return null;
  const direction = fill.side === "sell" ? -1 : 1;
  return (currentPrice - fill.price) * fill.quantity * direction;
}

function weightedAverage(values: number[], weights: number[]) {
  const totalWeight = weights.reduce((sum, value) => sum + value, 0);
  if (totalWeight <= 0) return 0;
  return values.reduce((sum, value, index) => sum + value * (weights[index] ?? 0), 0) / totalWeight;
}

function clamp(value: number, minimum: number, maximum: number) {
  return Math.min(maximum, Math.max(minimum, value));
}
