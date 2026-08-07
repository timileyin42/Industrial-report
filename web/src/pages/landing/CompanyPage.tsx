import { useState, type FormEvent } from "react";
import { ShieldCheck, BadgeCheck, Network } from "lucide-react";
import { LandingNav } from "./LandingNav";
import { LandingFooter } from "./LandingFooter";

// Reference: design/landing_company/code.html. "success-glance" /
// "warning-glance" from the mockup aren't real DESIGN.md tokens — mapped
// to primary/secondary respectively, consistent with this system's fixed
// status semantics (green=healthy, amber=warning) rather than inventing
// new colors.
export function CompanyPage() {
  const [submitted, setSubmitted] = useState(false);

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSubmitted(true);
  }

  return (
    <div className="bg-background text-on-background font-body-base antialiased min-h-screen flex flex-col">
      <LandingNav />
      <main className="flex-grow pt-32">
        <section className="relative min-h-[60vh] flex items-center border-b border-outline-variant">
          <div className="max-w-7xl mx-auto px-grid-margin relative z-10 w-full">
            <div className="lg:max-w-3xl py-12">
              <div className="inline-flex items-center gap-2 mb-6">
                <span className="bg-surface-variant text-on-surface px-2 py-1 rounded font-data-mono-sm text-data-mono-sm tracking-wider uppercase border border-outline-variant">
                  Mission Status
                </span>
                <span className="text-primary font-data-mono-sm text-data-mono-sm">ACTIVE</span>
              </div>
              <h1 className="font-headline-lg text-headline-lg text-on-surface mb-6">
                Providing the data infrastructure for the <span className="text-primary">African energy transition</span>.
              </h1>
              <p className="text-[18px] leading-relaxed text-on-surface-variant max-w-2xl">
                We build secure, scalable, and auditable data pipelines for mission-critical solar fleet
                operations — ensuring integrity from sensor to record.
              </p>
            </div>
          </div>
        </section>

        <section className="py-24">
          <div className="max-w-7xl mx-auto px-grid-margin">
            <div className="mb-16">
              <h2 className="font-headline-md text-headline-md text-on-surface mb-4">Core Principles</h2>
              <p className="text-on-surface-variant text-[18px] max-w-xl">
                Engineering-led infrastructure designed for absolute control and data integrity.
              </p>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-12 gap-gutter">
              <div className="md:col-span-8 bg-surface-container border border-outline-variant rounded p-8 hover:border-primary-container transition-colors">
                <div className="flex items-center mb-6">
                  <div className="w-10 h-10 bg-surface flex items-center justify-center border border-outline-variant rounded mr-4">
                    <ShieldCheck size={20} className="text-primary" />
                  </div>
                  <h3 className="font-headline-md text-headline-md text-on-surface">Zero-Trust Security</h3>
                </div>
                <p className="text-on-surface-variant font-body-base text-body-base mb-6 max-w-xl">
                  Our architecture operates on a strict zero-trust model. Every telemetry packet is
                  authenticated and verified before integration — designed for continuous operational
                  command even in adversarial environments.
                </p>
                <div className="bg-surface-dim border border-outline-variant rounded p-4 font-data-mono-sm text-data-mono-sm text-primary">
                  <div className="flex gap-2 mb-2 border-b border-outline-variant pb-2">
                    <div className="w-2 h-2 rounded-full bg-error" />
                    <div className="w-2 h-2 rounded-full bg-secondary" />
                    <div className="w-2 h-2 rounded-full bg-primary" />
                  </div>
                  <div>&gt; INIT SECURE_TUNNEL</div>
                  <div>&gt; AUTHENTICATING SENSOR_NODE_A7... <span className="text-primary">OK</span></div>
                  <div>&gt; ESTABLISHING E2E ENCRYPTION... <span className="text-primary">OK</span></div>
                  <div className="animate-pulse">&gt; AWAITING TELEMETRY_STREAM...</div>
                </div>
              </div>
              <div className="md:col-span-4 bg-surface-container border border-outline-variant rounded p-8 hover:border-primary-container transition-colors">
                <div className="w-10 h-10 bg-surface flex items-center justify-center border border-outline-variant rounded mb-6">
                  <BadgeCheck size={20} className="text-secondary" />
                </div>
                <h3 className="font-headline-md text-headline-md text-on-surface mb-4">Data Integrity</h3>
                <p className="text-on-surface-variant font-body-base text-body-base mb-8">
                  Append-only logging ensures historical data is audit-ready and tamper-evident.
                </p>
                <div className="space-y-4">
                  <div className="flex items-center justify-between border-b border-outline-variant pb-2">
                    <span className="font-data-mono-sm text-data-mono-sm text-on-surface-variant">RECORD HASH</span>
                    <span className="font-data-mono-sm text-data-mono-sm text-primary truncate w-24">0x7f8a9b...</span>
                  </div>
                  <div className="flex items-center justify-between border-b border-outline-variant pb-2">
                    <span className="font-data-mono-sm text-data-mono-sm text-on-surface-variant">VERIFICATION</span>
                    <span className="bg-primary/20 text-primary px-2 py-0.5 rounded text-[10px] font-bold">PASSED</span>
                  </div>
                </div>
              </div>
              <div className="md:col-span-12 bg-surface-container border border-outline-variant rounded p-8 hover:border-primary-container transition-colors">
                <div className="grid grid-cols-1 md:grid-cols-3 gap-8 items-center">
                  <div>
                    <div className="w-10 h-10 bg-surface flex items-center justify-center border border-outline-variant rounded mb-6">
                      <Network size={20} className="text-tertiary" />
                    </div>
                    <h3 className="font-headline-md text-headline-md text-on-surface mb-4">Infinite Scalability</h3>
                    <p className="text-on-surface-variant font-body-base text-body-base">
                      Distributed infrastructure built for the exponential growth of solar assets across
                      fragmented geographies.
                    </p>
                  </div>
                  <div className="col-span-2 grid grid-cols-2 md:grid-cols-4 gap-4">
                    {[
                      { label: "NODES", value: "12k+", border: "border-l-primary" },
                      { label: "UPTIME", value: "99.9%", border: "border-l-secondary" },
                      { label: "LATENCY", value: "<10ms", border: "border-l-tertiary" },
                      { label: "INGEST", value: "5TB/d", border: "border-l-primary" },
                    ].map((s) => (
                      <div key={s.label} className={`bg-surface p-4 border border-outline-variant border-l-2 ${s.border} rounded`}>
                        <div className="font-data-mono-sm text-data-mono-sm text-on-surface-variant mb-1">{s.label}</div>
                        <div className="font-data-display-lg text-data-display-lg text-on-surface">{s.value}</div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section className="py-24 border-t border-outline-variant bg-surface-container-low">
          <div className="max-w-3xl mx-auto px-grid-margin text-center">
            <h2 className="font-headline-lg text-headline-lg text-on-surface mb-6">Initialize Deployment</h2>
            <p className="text-[18px] leading-relaxed text-on-surface-variant mb-10">
              Partner with our engineering team to architect a robust data pipeline for your energy assets.
            </p>
            {submitted ? (
              <p className="font-label-caps text-label-caps text-primary">
                Request received — our team will reach out shortly.
              </p>
            ) : (
              <form className="space-y-4 max-w-md mx-auto text-left" onSubmit={handleSubmit}>
                <div>
                  <label className="block font-data-mono-sm text-data-mono-sm text-on-surface-variant mb-2">ORGANIZATION</label>
                  <input
                    required
                    className="w-full bg-surface border border-outline-variant rounded px-4 py-2 text-on-surface font-data-mono-sm text-data-mono-sm focus:border-primary focus:ring-1 focus:ring-primary outline-none"
                    placeholder="Company Name"
                    type="text"
                  />
                </div>
                <div>
                  <label className="block font-data-mono-sm text-data-mono-sm text-on-surface-variant mb-2">EMAIL</label>
                  <input
                    required
                    className="w-full bg-surface border border-outline-variant rounded px-4 py-2 text-on-surface font-data-mono-sm text-data-mono-sm focus:border-primary focus:ring-1 focus:ring-primary outline-none"
                    placeholder="user@domain.com"
                    type="email"
                  />
                </div>
                <button
                  type="submit"
                  className="w-full bg-primary-container text-on-primary-container px-4 py-3 rounded font-label-caps text-label-caps uppercase hover:bg-primary transition-colors mt-6"
                >
                  Request System Architecture Review
                </button>
              </form>
            )}
          </div>
        </section>
      </main>
      <LandingFooter />
    </div>
  );
}
