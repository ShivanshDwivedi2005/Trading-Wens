import type { MarketSnapshot } from "./market-api";
import type { PriceMovementPrediction } from "./market-prediction";
import type { NewsArticle } from "./news-api";
import type { TradingPosition } from "./trading-api";

export function positionRisk(
  position: TradingPosition,
  portfolioValue: number,
  prediction?: PriceMovementPrediction,
) {
  const weight = portfolioValue > 0 ? (Math.abs(position.market_value) / portfolioValue) * 100 : 0;
  const downside = Math.max(0, -position.change_today);
  const loss = Math.max(0, -position.unrealized_pl_percent);
  const isShort = position.side.toLowerCase() === "short" || position.quantity < 0;
  const adverse = prediction
    ? (isShort && prediction.direction === "up") || (!isShort && prediction.direction === "down")
    : false;
  const aligned = prediction
    ? (isShort && prediction.direction === "down") || (!isShort && prediction.direction === "up")
    : false;
  const directionalStrength = prediction
    ? Math.max(0, (prediction.possibility - 50) / 50) * (prediction.confidence / 100)
    : 0;
  const baseScore = weight * 0.06 + downside * 0.55 + loss * 0.15;
  const score = Math.min(
    10,
    Math.max(
      0,
      adverse ? baseScore + directionalStrength * 6 : aligned ? baseScore * 0.35 : baseScore,
    ),
  );
  const level = score >= 7.5 ? "High" : score >= 4.5 ? "Moderate" : "Low";
  const confidenceFields = [
    position.current_price,
    position.market_value,
    position.cost_basis,
    position.average_entry_price,
  ].filter((value) => Number.isFinite(value) && value !== 0).length;
  const confidence = Math.round((confidenceFields / 4) * 100);
  const action = adverse
    ? `Risk: ${isShort ? "upward" : "downward"} move conflicts with ${isShort ? "short" : "long"} position`
    : aligned
      ? `Aligned: predicted ${prediction?.label.toLowerCase()} supports ${isShort ? "short" : "long"} position`
      : weight > 35
        ? "Review concentration"
        : position.change_today < -3
          ? "Review downside limits"
          : position.unrealized_pl_percent < -5
            ? "Reassess position thesis"
            : "Monitor position";
  return {
    score,
    level,
    confidence,
    action,
    weight,
    alignment: adverse
      ? ("adverse" as const)
      : aligned
        ? ("aligned" as const)
        : ("neutral" as const),
  };
}

export function portfolioRisk(
  positions: TradingPosition[],
  portfolioValue: number,
  predictions: Map<string, PriceMovementPrediction> = new Map(),
) {
  if (!positions.length || portfolioValue <= 0) {
    return {
      score: 0,
      level: "Low",
      concentration: 0,
      downside: 0,
      adverseCount: 0,
      alignedCount: 0,
    };
  }
  const metrics = positions.map((position) =>
    positionRisk(position, portfolioValue, predictions.get(position.symbol)),
  );
  const concentration = Math.max(...metrics.map((metric) => metric.weight));
  const investedValue = positions.reduce(
    (sum, position) => sum + Math.abs(position.market_value),
    0,
  );
  const score = metrics.reduce(
    (sum, metric, index) =>
      sum +
      metric.score *
        (investedValue > 0 ? Math.abs(positions[index]?.market_value ?? 0) / investedValue : 0),
    0,
  );
  const downside = metrics.reduce(
    (sum, metric, index) =>
      sum +
      (metric.alignment === "adverse" ? metric.score : 0) *
        (investedValue > 0 ? Math.abs(positions[index]?.market_value ?? 0) / investedValue : 0),
    0,
  );
  return {
    score,
    level: score >= 7.5 ? "High" : score >= 4.5 ? "Moderate" : "Low",
    concentration,
    downside,
    adverseCount: metrics.filter((metric) => metric.alignment === "adverse").length,
    alignedCount: metrics.filter((metric) => metric.alignment === "aligned").length,
  };
}

export function sentimentWatchlist(snapshots: MarketSnapshot[], articles: NewsArticle[]) {
  return snapshots
    .map((snapshot) => {
      const related = articles.filter((article) =>
        article.matched_symbols.includes(snapshot.symbol),
      );
      const analyzed = related.flatMap((article) => (article.sentiment ? [article.sentiment] : []));
      const score = analyzed.length
        ? analyzed.reduce((sum, sentiment) => sum + sentiment.score, 0) / analyzed.length
        : null;
      const confidence = analyzed.length
        ? Math.round(
            (analyzed.reduce((sum, sentiment) => sum + sentiment.confidence, 0) / analyzed.length) *
              100,
          )
        : 0;
      return {
        snapshot,
        score,
        confidence,
        articleCount: related.length,
        analyzedCount: analyzed.length,
      };
    })
    .sort((a, b) => (b.score ?? -2) - (a.score ?? -2) || b.articleCount - a.articleCount)
    .slice(0, 5);
}
