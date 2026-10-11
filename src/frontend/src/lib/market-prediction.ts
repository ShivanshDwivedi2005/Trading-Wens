import type { MarketSnapshot } from "./market-api";
import type { NewsArticle } from "./news-api";

export type MovementDirection = "up" | "down" | "sideways";

export type PriceMovementPrediction = {
  direction: MovementDirection;
  label: "Move up" | "Move down" | "Sideways";
  possibility: number;
  confidence: number;
  score: number;
  analyzedHeadlines: number;
  matchedHeadlines: number;
  reason: string;
};

export function priceMovementPrediction(
  snapshot: Pick<
    MarketSnapshot,
    "symbol" | "price" | "change_percent" | "high" | "low" | "available"
  >,
  articles: NewsArticle[],
): PriceMovementPrediction {
  const related = articles.filter((article) => article.matched_symbols.includes(snapshot.symbol));
  const analyzed = related.flatMap((article) => (article.sentiment ? [article.sentiment] : []));
  const sentiment = analyzed.length
    ? weightedAverage(
        analyzed.map((item) => item.score),
        analyzed.map((item) => item.confidence),
      )
    : 0;
  const momentum = snapshot.available ? clamp(snapshot.change_percent / 5, -1, 1) : 0;
  const range = snapshot.high - snapshot.low;
  const rangePosition =
    snapshot.available && range > 0
      ? clamp(((snapshot.price - snapshot.low) / range - 0.5) * 2, -1, 1)
      : 0;
  const score = momentum * 0.45 + rangePosition * 0.2 + sentiment * 0.35;
  const direction: MovementDirection = score > 0.08 ? "up" : score < -0.08 ? "down" : "sideways";
  const possibility =
    direction === "sideways"
      ? Math.round(55 + (1 - Math.min(1, Math.abs(score) * 4)) * 20)
      : Math.round(50 + Math.min(0.45, Math.abs(score)) * 100);
  const averageModelConfidence = analyzed.length
    ? analyzed.reduce((sum, item) => sum + item.confidence, 0) / analyzed.length
    : 0;
  const marketEvidence = snapshot.available ? 0.5 : 0.15;
  const confidence = Math.round(
    clamp(
      marketEvidence + Math.min(analyzed.length, 5) * 0.045 + averageModelConfidence * 0.25,
      0,
      0.95,
    ) * 100,
  );
  const label = direction === "up" ? "Move up" : direction === "down" ? "Move down" : "Sideways";
  const newsReason = analyzed.length
    ? `${analyzed.length} trained-model headline${analyzed.length === 1 ? "" : "s"} contribute ${signed(sentiment)} sentiment.`
    : related.length
      ? `${related.length} matched headline${related.length === 1 ? " is" : "s are"} awaiting trained-model analysis.`
      : "No current model-scored headline matched this stock.";

  return {
    direction,
    label,
    possibility,
    confidence,
    score,
    analyzedHeadlines: analyzed.length,
    matchedHeadlines: related.length,
    reason: `Today's market move is ${signed(snapshot.change_percent)}%. ${newsReason}`,
  };
}

function weightedAverage(values: number[], weights: number[]) {
  const totalWeight = weights.reduce((sum, value) => sum + value, 0);
  if (totalWeight <= 0) return 0;
  return values.reduce((sum, value, index) => sum + value * (weights[index] ?? 0), 0) / totalWeight;
}

function signed(value: number) {
  return `${value >= 0 ? "+" : ""}${value.toFixed(2)}`;
}

function clamp(value: number, minimum: number, maximum: number) {
  return Math.min(maximum, Math.max(minimum, value));
}
