import { createFileRoute } from "@tanstack/react-router";
import { RiskWorkspace } from "@/components/market/RiskWorkspace";
export const Route = createFileRoute("/demo")({
  head: () => ({
    meta: [
      { title: "Live Demo — Trading wens" },
      {
        name: "description",
        content:
          "Explore a simulated live financial risk feed, sentiment charts, market events, and portfolio scenarios without signing in.",
      },
      { property: "og:title", content: "Live Demo — Trading wens" },
      {
        property: "og:description",
        content:
          "Explore simulated financial risk signals and portfolio scenarios without signing in.",
      },
      { property: "og:type", content: "website" },
      { name: "twitter:card", content: "summary_large_image" },
    ],
  }),
  component: () => <RiskWorkspace demo />,
});
