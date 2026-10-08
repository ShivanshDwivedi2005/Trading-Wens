import { createFileRoute } from "@tanstack/react-router";
import { OrderMonitor } from "@/components/market/OrderMonitor";

export const Route = createFileRoute("/_authenticated/orders")({
  validateSearch: (search: Record<string, unknown>) => ({
    symbol: typeof search["symbol"] === "string" ? search["symbol"].toUpperCase() : "",
  }),
  head: () => ({
    meta: [
      { title: "Order Monitor — Trading wens" },
      {
        name: "description",
        content:
          "Monitor Alpaca paper orders, fills, position evidence, and execution audit events.",
      },
    ],
  }),
  component: OrderMonitorRoute,
});

function OrderMonitorRoute() {
  const { symbol } = Route.useSearch();
  return <OrderMonitor initialSymbol={symbol} />;
}
