export type RiskEvent = {
  id: number;
  source: string;
  time: string;
  ticker: string;
  company: string;
  headline: string;
  excerpt: string;
  sentiment: number;
  eventType: string;
  impact: number;
  confidence: number;
  severity: "Critical" | "High" | "Moderate" | "Low";
  exposure: string;
  direction: string;
};

export const initialEvents: RiskEvent[] = [
  {
    id: 1,
    source: "GLOBAL NEWS",
    time: "JUST NOW",
    ticker: "NVDA",
    company: "NVIDIA Corp.",
    headline: "Export controls on advanced semiconductor chips tighten across key markets",
    excerpt:
      "New restrictions could reduce near-term data center revenue and introduce supply-chain uncertainty across the semiconductor sector.",
    sentiment: -0.78,
    eventType: "Regulatory / Geopolitical",
    impact: 9.2,
    confidence: 94,
    severity: "Critical",
    exposure: "$248,400",
    direction: "Downside risk",
  },
  {
    id: 2,
    source: "MARKET WIRE",
    time: "2 MIN AGO",
    ticker: "AAPL",
    company: "Apple Inc.",
    headline: "Apple supplier reports stronger-than-expected component demand",
    excerpt:
      "Revised production forecasts suggest improving device demand ahead of the next earnings cycle.",
    sentiment: 0.64,
    eventType: "Supply Chain",
    impact: 6.4,
    confidence: 88,
    severity: "Moderate",
    exposure: "$156,200",
    direction: "Upside potential",
  },
  {
    id: 3,
    source: "SOCIAL SIGNAL",
    time: "5 MIN AGO",
    ticker: "TSLA",
    company: "Tesla Inc.",
    headline: "EV pricing speculation accelerates across investor conversations",
    excerpt:
      "A sharp rise in social mentions points to potential margin pressure as competitive pricing intensifies.",
    sentiment: -0.42,
    eventType: "Market Sentiment",
    impact: 7.1,
    confidence: 82,
    severity: "High",
    exposure: "$92,800",
    direction: "Downside risk",
  },
  {
    id: 4,
    source: "EARNINGS DESK",
    time: "8 MIN AGO",
    ticker: "MSFT",
    company: "Microsoft Corp.",
    headline: "Cloud infrastructure revenue outlook revised higher by analysts",
    excerpt:
      "Improving enterprise AI spending expectations lift the forward outlook for cloud infrastructure.",
    sentiment: 0.82,
    eventType: "Earnings Outlook",
    impact: 8.3,
    confidence: 96,
    severity: "Low",
    exposure: "$184,600",
    direction: "Upside potential",
  },
  {
    id: 5,
    source: "GLOBAL NEWS",
    time: "12 MIN AGO",
    ticker: "JPM",
    company: "JPMorgan Chase",
    headline: "Central bank signals prolonged restrictive rate environment",
    excerpt:
      "Policy commentary raises uncertainty for credit conditions and rate-sensitive portfolios.",
    sentiment: -0.56,
    eventType: "Macroeconomic",
    impact: 7.8,
    confidence: 91,
    severity: "High",
    exposure: "$113,500",
    direction: "Downside risk",
  },
  {
    id: 6,
    source: "MARKET WIRE",
    time: "16 MIN AGO",
    ticker: "AMZN",
    company: "Amazon.com Inc.",
    headline: "Retail logistics efficiencies improve operating margin outlook",
    excerpt:
      "Distribution network improvements may support stronger margins into the coming quarter.",
    sentiment: 0.47,
    eventType: "Operational",
    impact: 5.9,
    confidence: 85,
    severity: "Low",
    exposure: "$78,900",
    direction: "Upside potential",
  },
];

export const marketIndices = [
  { name: "S&P 500", value: "5,842.47", change: "+0.84%", positive: true },
  { name: "NASDAQ", value: "18,394.21", change: "+1.26%", positive: true },
  { name: "DOW JONES", value: "42,118.63", change: "-0.18%", positive: false },
  { name: "VIX", value: "17.34", change: "+2.41%", positive: false },
];
export const sentimentHistory = [
  { hour: "09:00", NVDA: 18, AAPL: 48, MSFT: 43 },
  { hour: "10:00", NVDA: 32, AAPL: 43, MSFT: 51 },
  { hour: "11:00", NVDA: 26, AAPL: 53, MSFT: 48 },
  { hour: "12:00", NVDA: 41, AAPL: 58, MSFT: 61 },
  { hour: "13:00", NVDA: 35, AAPL: 54, MSFT: 56 },
  { hour: "14:00", NVDA: 29, AAPL: 65, MSFT: 71 },
  { hour: "15:00", NVDA: 47, AAPL: 61, MSFT: 68 },
  { hour: "16:00", NVDA: 34, AAPL: 72, MSFT: 77 },
  { hour: "17:00", NVDA: 24, AAPL: 68, MSFT: 82 },
];
export const severityRank = { Critical: 4, High: 3, Moderate: 2, Low: 1 };
