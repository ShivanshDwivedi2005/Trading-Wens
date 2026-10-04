import { useEffect, useRef } from "react";

type Candle = { open: number; high: number; low: number; close: number };

// Synthetic price history for a decorative chart; no live market data is represented.
function makeCandles(count: number): Candle[] {
  let seed = 18731;
  const random = () => {
    seed = (seed * 16807) % 2147483647;
    return seed / 2147483647;
  };
  const candles: Candle[] = [];
  let price = 124;
  for (let i = 0; i < count; i++) {
    const progress = i / count;
    const drift =
      progress > 0.68 && progress < 0.82 ? -1.9 : progress > 0.82 && progress < 0.94 ? 2.2 : 0.16;
    const open = price;
    const close = Math.max(25, open + (random() - 0.49) * 6.8 + drift);
    candles.push({
      open,
      close,
      high: Math.max(open, close) + random() * 3.3 + 0.5,
      low: Math.min(open, close) - random() * 3.3 - 0.5,
    });
    price = close;
  }
  return candles;
}

export function MarketScene() {
  const ref = useRef<HTMLCanvasElement>(null);
  useEffect(() => {
    const canvas = ref.current;
    if (!canvas) return;
    const context = canvas.getContext("2d");
    if (!context) return;
    let width = 0;
    let height = 0;
    let animation = 0;
    const candles = makeCandles(180);
    const movements = candles.map((_, index) => ({
      phase: index * 2.37,
      speed: 0.65 + (index % 9) * 0.095,
      amplitude: 3 + (index % 7) * 0.52,
    }));
    const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const resize = () => {
      const rect = canvas.getBoundingClientRect();
      const ratio = Math.min(window.devicePixelRatio || 1, 2);
      width = rect.width;
      height = rect.height;
      canvas.width = Math.round(width * ratio);
      canvas.height = Math.round(height * ratio);
      context.setTransform(ratio, 0, 0, ratio, 0, 0);
      draw(0);
    };
    const draw = (seconds: number) => {
      if (!width || !height) return;
      const styles = getComputedStyle(canvas);
      const grid = styles.getPropertyValue("--chart-grid").trim();
      const rise = styles.getPropertyValue("--chart-rise").trim();
      const fall = styles.getPropertyValue("--chart-fall").trim();
      const axis = styles.getPropertyValue("--chart-axis").trim();
      context.clearRect(0, 0, width, height);
      const step = width < 650 ? 10 : 9;
      const visibleCount = Math.ceil(width / step);
      const visible = candles.slice(-visibleCount);
      const lowest = Math.min(...visible.map((c) => c.low)) - 14;
      const highest = Math.max(...visible.map((c) => c.high)) + 14;
      const range = Math.max(highest - lowest, 35);
      const y = (price: number) => 48 + ((highest - price) / range) * Math.max(height - 120, 1);
      context.strokeStyle = grid;
      context.lineWidth = 1;
      for (let gx = 0; gx < width; gx += 76) {
        context.beginPath();
        context.moveTo(gx + 0.5, 0);
        context.lineTo(gx + 0.5, height);
        context.stroke();
      }
      for (let gy = 48; gy < height; gy += 62) {
        context.beginPath();
        context.moveTo(0, gy + 0.5);
        context.lineTo(width, gy + 0.5);
        context.stroke();
      }
      visible.forEach((candle, index) => {
        const x = width - (visible.length - 1 - index) * step - step / 2;
        const movement = movements[candles.length - visible.length + index];
        if (!movement) return;
        const close =
          candle.open +
          (candle.close - candle.open) * 0.55 +
          Math.sin(seconds * movement.speed + movement.phase) * movement.amplitude;
        const upperWick = candle.high - Math.max(candle.open, candle.close);
        const lowerWick = Math.min(candle.open, candle.close) - candle.low;
        const color = close >= candle.open ? rise : fall;
        context.strokeStyle = color;
        context.fillStyle = color;
        context.lineWidth = 1;
        context.beginPath();
        context.moveTo(x + 0.5, y(Math.max(candle.open, close) + upperWick));
        context.lineTo(x + 0.5, y(Math.min(candle.open, close) - lowerWick));
        context.stroke();
        const top = Math.min(y(candle.open), y(close));
        context.fillRect(x - 2.5, top, 5, Math.max(2, Math.abs(y(close) - y(candle.open))));
      });
      const latest = visible[visible.length - 1];
      const latestMovement = movements[candles.length - 1];
      if (latest && latestMovement) {
        const liveClose =
          latest.open +
          (latest.close - latest.open) * 0.55 +
          Math.sin(seconds * latestMovement.speed + latestMovement.phase) *
            latestMovement.amplitude;
        const lastY = y(liveClose);
        context.strokeStyle = axis;
        context.setLineDash([4, 6]);
        context.beginPath();
        context.moveTo(0, lastY);
        context.lineTo(width, lastY);
        context.stroke();
        context.setLineDash([]);
      }
    };
    const tick = (time: number) => {
      draw(time / 1000);
      animation = requestAnimationFrame(tick);
    };
    resize();
    window.addEventListener("resize", resize);
    if (!reduceMotion) animation = requestAnimationFrame(tick);
    return () => {
      cancelAnimationFrame(animation);
      window.removeEventListener("resize", resize);
    };
  }, []);
  return <canvas ref={ref} className="absolute inset-0 h-full w-full" aria-hidden="true" />;
}
