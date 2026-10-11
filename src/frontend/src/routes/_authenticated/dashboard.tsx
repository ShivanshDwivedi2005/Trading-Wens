import { createFileRoute } from "@tanstack/react-router";
import { RiskWorkspace } from "@/components/market/RiskWorkspace";
export const Route = createFileRoute("/_authenticated/dashboard")({
  head: () => ({
    meta: [
      { title: "Workspace — Trading wens" },
      {
        name: "description",
        content:
          "Monitor market risk signals, sentiment, portfolio exposure, and stress scenarios in your Trading wens workspace.",
      },
      { property: "og:title", content: "Workspace — Trading wens" },
      { property: "og:description", content: "Your financial risk intelligence workspace." },
      { property: "og:type", content: "website" },
      { name: "twitter:card", content: "summary_large_image" },
    ],
  }),
  component: () => <RiskWorkspace />,
});
