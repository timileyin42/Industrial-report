// Drops the current, still-in-progress period from a bucketed trend
// series. Without this, a multi-period comparison chart (Energy Output,
// Specific Yield, Performance Ratio, a monthly Year view, etc.) always
// shows a fake-looking drop on its most recent point — not a real
// decline, just the current period's total not being finished yet next
// to a prior period's completed total.
//
// Bucketing is UTC (this platform's storage/comparison convention), so
// "now" is compared as a UTC calendar date/month, matching how the
// buckets themselves are computed server-side.
//
// Only for multi-period trend charts. The current period's own live
// figures (the Dashboard's "Energy Today" KPI, its intraday Day-view
// power curve) deliberately still show today's real, growing value
// elsewhere — this only affects charts that plot one point per
// completed period.
export function excludeInProgressPeriod<T extends { period_start: string }>(
  points: T[],
  period: "daily" | "monthly" = "daily",
): T[] {
  const nowUTC = new Date().toISOString();
  const currentKey = period === "monthly" ? nowUTC.slice(0, 7) : nowUTC.slice(0, 10);
  const keyLen = currentKey.length;
  return points.filter((p) => p.period_start.slice(0, keyLen) !== currentKey);
}

// Convenience alias for the common daily case.
export function excludeInProgressToday<T extends { period_start: string }>(points: T[]): T[] {
  return excludeInProgressPeriod(points, "daily");
}
