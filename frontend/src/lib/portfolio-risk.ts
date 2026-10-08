import type { MarketSnapshot } from "./market-api";
import type { NewsArticle } from "./news-api";
import type { TradingPosition } from "./trading-api";

export function positionRisk(position: TradingPosition, portfolioValue: number) {
  const weight = portfolioValue > 0 ? (Math.abs(position.market_value) / portfolioValue) * 100 : 0;
  const downside = Math.max(0, -position.change_today);
  const loss = Math.max(0, -position.unrealized_pl_percent);
  const score = Math.min(10, Math.max(1, weight * 0.08 + downside * 0.75 + loss * 0.18));
  const level = score >= 7.5 ? "High" : score >= 4.5 ? "Moderate" : "Low";
  const confidenceFields = [
    position.current_price,
    position.market_value,
    position.cost_basis,
    position.average_entry_price,
  ].filter((value) => Number.isFinite(value) && value !== 0).length;
  const confidence = Math.round((confidenceFields / 4) * 100);
  const action =
    weight > 35
      ? "Review concentration"
      : position.change_today < -3
        ? "Review downside limits"
        : position.unrealized_pl_percent < -5
          ? "Reassess position thesis"
          : "Monitor position";
  return { score, level, confidence, action, weight };
}

export function portfolioRisk(positions: TradingPosition[], portfolioValue: number) {
  if (!positions.length || portfolioValue <= 0) {
    return { score: 1, level: "Low", concentration: 0, downside: 0 };
  }
  const metrics = positions.map((position) => positionRisk(position, portfolioValue));
  const concentration = Math.max(...metrics.map((metric) => metric.weight));
  const downside = positions.reduce(
    (sum, position) =>
      sum +
      Math.max(0, -position.change_today) * (Math.abs(position.market_value) / portfolioValue),
    0,
  );
  const score = Math.min(10, Math.max(1, concentration * 0.09 + downside * 0.9));
  return {
    score,
    level: score >= 7.5 ? "High" : score >= 4.5 ? "Moderate" : "Low",
    concentration,
    downside,
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
        : 0;
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
    .sort((a, b) => b.score - a.score || b.articleCount - a.articleCount)
    .slice(0, 5);
}
