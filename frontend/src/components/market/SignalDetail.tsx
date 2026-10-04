import {
  ArrowDownRight,
  ArrowUpRight,
  BrainCircuit,
  CheckCircle2,
  CircleDot,
  ScanSearch,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import type { RiskEvent } from "@/lib/market";

export function SignalDetail({ event, onClose }: { event: RiskEvent; onClose: () => void }) {
  return (
    <div className="signal-detail animate-in fade-in slide-in-from-right-2 duration-300">
      <div className="flex items-start justify-between gap-4 border-b border-border pb-5">
        <div>
          <div className="label text-primary">
            SIGNAL INTELLIGENCE / {String(event.id).padStart(4, "0")}
          </div>
          <h2 className="mt-2 text-xl font-semibold text-foreground">Event analysis</h2>
        </div>
        <Button size="icon" variant="ghost" aria-label="Close analysis" onClick={onClose}>
          <X />
        </Button>
      </div>
      <div className="mt-6 flex items-center gap-3">
        <div className="flex size-11 items-center justify-center rounded-md border border-primary/30 bg-primary/10 text-primary">
          <BrainCircuit size={20} />
        </div>
        <div>
          <p className="font-semibold">{event.company}</p>
          <p className="text-xs text-muted-foreground">
            {event.ticker} · {event.source}
          </p>
        </div>
      </div>
      <h3 className="mt-7 text-lg font-medium leading-snug">{event.headline}</h3>
      <p className="mt-3 text-sm leading-7 text-muted-foreground">{event.excerpt}</p>
      <div className="mt-7 border-y border-border py-5">
        <div className="label text-muted-foreground">RAW INPUT → STRUCTURED SIGNAL</div>
        <div className="mt-5 space-y-5">
          <div className="flex gap-3">
            <ScanSearch size={17} className="mt-0.5 text-primary" />
            <div>
              <div className="text-xs font-semibold">Entity identified</div>
              <div className="mt-1 text-sm text-muted-foreground">
                {event.company} ({event.ticker})
              </div>
            </div>
          </div>
          <div className="flex gap-3">
            <CircleDot size={17} className="mt-0.5 text-primary" />
            <div>
              <div className="text-xs font-semibold">Event classified</div>
              <div className="mt-1 text-sm text-muted-foreground">{event.eventType}</div>
            </div>
          </div>
          <div className="flex gap-3">
            <CheckCircle2 size={17} className="mt-0.5 text-primary" />
            <div>
              <div className="text-xs font-semibold">Risk signal generated</div>
              <div className="mt-1 text-sm text-muted-foreground">
                {event.direction} · {event.confidence}% confidence
              </div>
            </div>
          </div>
        </div>
      </div>
      <div className="mt-6 grid grid-cols-3 gap-2">
        {[
          [
            "SENTIMENT",
            event.sentiment > 0 ? `+${event.sentiment.toFixed(2)}` : event.sentiment.toFixed(2),
          ],
          ["IMPACT", `${event.impact}/10`],
          ["CONFIDENCE", `${event.confidence}%`],
        ].map(([label, value]) => (
          <div key={label} className="rounded-md border border-border bg-secondary/40 p-3">
            <div className="label text-muted-foreground">{label}</div>
            <div className="mt-2 text-lg font-semibold tabular-nums">{value}</div>
          </div>
        ))}
      </div>
      <div className="mt-6 rounded-md border border-border bg-secondary/30 p-5">
        <div className="label text-muted-foreground">PORTFOLIO EXPOSURE</div>
        <div className="mt-3 flex items-end justify-between">
          <div className="text-2xl font-semibold tabular-nums">{event.exposure}</div>
          <div
            className={
              event.sentiment < 0
                ? "flex items-center gap-1 text-xs text-destructive"
                : "flex items-center gap-1 text-xs text-primary"
            }
          >
            {event.sentiment < 0 ? <ArrowDownRight size={15} /> : <ArrowUpRight size={15} />}
            {event.direction}
          </div>
        </div>
        <p className="mt-4 text-xs leading-5 text-muted-foreground">
          Illustrative position exposure. Simulated data is not investment advice.
        </p>
      </div>
    </div>
  );
}
