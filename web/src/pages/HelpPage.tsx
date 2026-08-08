import { TopNav } from "../components/layout/TopNav";

// Real, accurate operator-facing guidance — not marketing copy. Mirrors
// the honesty of docs/demo-guide.md, just written for an operator using
// the dashboard rather than someone demoing the backend.
const SECTIONS: { title: string; body: string }[] = [
  {
    title: "Getting an account",
    body:
      "There's no public sign-up — this dashboard is invite-only. An operator invites new users from Settings → Invite User; " +
      "the invitee gets a real email with a link to set their own password.",
  },
  {
    title: "Operator vs. restricted access",
    body:
      "An operator sees the whole fleet. A restricted account is locked to exactly one site, enforced on every request — " +
      "not just hidden in this interface, so it can't be bypassed by editing a URL.",
  },
  {
    title: "What the status colors mean",
    body: "Green = online/optimal, amber = degraded/warning, red = offline/fault. These three colors mean the same thing everywhere in the dashboard.",
  },
  {
    title: '"Not tracked" / "—" values',
    body:
      "Some fields (like battery storage or grid import on the Energy Flow panel) show \"Not tracked\" because this platform has no data source for them yet — " +
      "never a fabricated number standing in for a real one.",
  },
  {
    title: "Current Generation vs. Energy Today",
    body:
      "Current Generation is a live, right-now figure (the latest reading from every online device, summed). Energy Today is cumulative — how much has been generated so far today. " +
      "They answer different questions and won't match.",
  },
  {
    title: "Capacity Factor vs. Performance Ratio",
    body:
      "Capacity Factor compares output to a flat theoretical maximum with no weather adjustment — it's naturally low and varies by season, even for a healthy system. " +
      "Performance Ratio compares output to what the site should have produced given the sunlight it actually received that day — a real fault/health signal, since it stays fairly steady regardless of weather.",
  },
  {
    title: "Reviewing a past date",
    body: "The date picker on the Dashboard anchors the Generation Overview chart and the day-over-day KPI comparisons to that day. Current Generation and Top Performing Sites always reflect right now / today, since neither concept has a meaningful past-date equivalent.",
  },
  {
    title: "Emission factors and CO2 figures",
    body:
      "CO2-avoided figures need a grid emission factor set first (Settings → Emission Factor) — this platform never guesses one. Each site resolves its own factor from its own country, " +
      "and a fleet spanning more than one country reports each grid's contribution separately rather than blending them into one number.",
  },
];

export function HelpPage() {
  return (
    <>
      <TopNav title="Help" />
      <div className="flex-1 p-grid-margin max-w-2xl space-y-4">
        {SECTIONS.map((s) => (
          <div key={s.title} className="glass-card rounded-2xl p-5">
            <h3 className="font-headline-md text-[15px] font-bold text-on-surface mb-1.5">{s.title}</h3>
            <p className="text-on-surface-variant text-[13px] leading-relaxed">{s.body}</p>
          </div>
        ))}
        <div className="glass-card rounded-2xl p-5">
          <h3 className="font-headline-md text-[15px] font-bold text-on-surface mb-1.5">Still need help?</h3>
          <p className="text-on-surface-variant text-[13px] leading-relaxed">
            Reach your organization's operator, or contact{" "}
            <a href="mailto:support@cleanenergyanalytics.co.uk" className="text-primary underline">
              support@cleanenergyanalytics.co.uk
            </a>
            .
          </p>
        </div>
      </div>
    </>
  );
}
