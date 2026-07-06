/**
 * Muted fallback shown when a chart has no data to plot. Charts must degrade
 * gracefully in this offline-first PWA: an empty dataset (offline cache miss
 * or an empty period) renders this message instead of a blank chart canvas.
 */
export function ChartEmpty({ message = 'Sem dados no período' }: { message?: string }) {
  return (
    <p
      role="status"
      className="flex h-40 items-center justify-center text-sm text-gray-500"
    >
      {message}
    </p>
  );
}
