import type { NewsSentiment } from "@/lib/news-api";

export function SentimentBadge({ sentiment }: { sentiment: NewsSentiment | undefined }) {
  if (!sentiment) {
    return <span className="sentiment-badge sentiment-unavailable">Analysis unavailable</span>;
  }

  const score = `${sentiment.score >= 0 ? "+" : ""}${sentiment.score.toFixed(2)}`;
  const confidence = `${Math.round(sentiment.confidence * 100)}% confidence`;
  return (
    <span
      className={`sentiment-badge sentiment-${sentiment.label.toLowerCase()}`}
      aria-label={`${sentiment.label.toLowerCase()} sentiment, score ${score}, ${confidence}`}
    >
      <span>{sentiment.label}</span>
      <span aria-hidden="true">·</span>
      <span className="tabular-nums">{score}</span>
      <span aria-hidden="true">·</span>
      <span className="tabular-nums">{confidence}</span>
    </span>
  );
}
