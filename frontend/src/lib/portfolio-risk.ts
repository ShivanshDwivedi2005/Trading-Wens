import type { MarketSnapshot } from "./market-api";
import type { NewsArticle } from "./news-api";
import type { TradingPosition } from "./trading-api";

const positiveWords = [
  "beat",
  "beats",
  "growth",
  "gain",
  "gains",
  "record",
  "upgrade",
  "strong",
  "surge",
  "expands",
  "rises",
];

const negativeWords = [
  "miss",
  "misses",
  "decline",
  "loss",
  "cuts",
  "downgrade",
  "weak",
  "falls",
  "risk",
  "probe",
  "lawsuit",
];

export function headlineSentiment(title: string) {
  const words = title.toLowerCase().split(/[^a-z]+/);
  const raw = words.reduce(
    (score, word) =>
      score + (positiveWords.includes(word) ? 1 : 0) - (negativeWords.includes(word) ? 1 : 0),
    0,
  );
  return Math.max(-1, Math.min(1, raw / 3));
}

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
      const score = related.length
        ? related.reduce((sum, article) => sum + headlineSentiment(article.title), 0) /
          related.length
        : 0;
      const confidence = Math.min(95, 35 + related.length * 12);
      return { snapshot, score, confidence, articleCount: related.length };
    })
    .sort((a, b) => b.score - a.score || b.articleCount - a.articleCount)
    .slice(0, 5);
}
