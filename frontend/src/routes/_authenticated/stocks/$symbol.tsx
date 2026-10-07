import { createFileRoute } from "@tanstack/react-router";
import { StockDetailPage } from "@/components/market/StockDetailPage";

export const Route = createFileRoute("/_authenticated/stocks/$symbol")({
  head: ({ params }) => ({
    meta: [
      { title: `${params.symbol.toUpperCase()} market detail — Trading wens` },
      {
        name: "description",
        content: `Live price history and recent company news for ${params.symbol.toUpperCase()}.`,
      },
    ],
  }),
  component: StockRoute,
});

function StockRoute() {
  const { symbol } = Route.useParams();
  return <StockDetailPage symbol={symbol.toUpperCase()} />;
}
