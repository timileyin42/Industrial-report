import { SunMedium, Landmark, Zap } from "lucide-react";
import { LandingNav } from "./LandingNav";
import { LandingFooter } from "./LandingFooter";

// Reference: design/landing_solutions/code.html. The mockup's three
// decorative background images were literal "placeholder" strings in the
// Stitch export (no real URL was ever generated for them) — omitted
// rather than inventing a stand-in image for something that was never
// actually there.
export function SolutionsPage() {
  return (
    <div className="bg-background text-on-background antialiased min-h-screen flex flex-col">
      <LandingNav />
      <main className="flex-grow w-full max-w-7xl mx-auto px-grid-margin py-24 space-y-24 pt-32">
        <section className="flex flex-col items-center text-center space-y-6">
          <h1 className="font-headline-lg text-headline-lg text-on-surface max-w-4xl">
            Infrastructure Built for <span className="text-primary">Scale &amp; Integrity</span>
          </h1>
          <p className="text-[18px] leading-relaxed text-on-surface-variant max-w-2xl">
            Clean Energy Analytics delivers real-time telemetry and audit-ready data tailored for the specialized
            needs of modern solar operators.
          </p>
        </section>

        <section className="grid grid-cols-1 md:grid-cols-6 gap-gutter">
          <div className="md:col-span-3 bg-surface-container border border-outline-variant rounded-lg p-8 hover:border-primary-container transition-colors">
            <div className="flex items-center gap-3 mb-6">
              <SunMedium size={22} className="text-primary" />
              <h2 className="font-headline-md text-headline-md text-on-surface">Solar Installers</h2>
            </div>
            <p className="font-body-base text-body-base text-on-surface-variant mb-6">
              Manage explosive fleet growth with automated device lifecycle management. Reduce truck rolls
              through remote diagnostics.
            </p>
            <div className="space-y-4 pt-4 border-t border-outline-variant">
              <ul className="space-y-2 font-body-base text-body-base text-on-surface-variant">
                <li className="flex items-center gap-2"><span className="text-primary">✓</span> Zero-Touch Provisioning</li>
                <li className="flex items-center gap-2"><span className="text-primary">✓</span> OTA Firmware Management</li>
              </ul>
            </div>
          </div>

          <div className="md:col-span-3 bg-surface-container border border-outline-variant rounded-lg p-8 hover:border-primary-container transition-colors">
            <div className="flex items-center gap-3 mb-6">
              <Landmark size={22} className="text-secondary" />
              <h2 className="font-headline-md text-headline-md text-on-surface">Asset Owners</h2>
            </div>
            <p className="font-body-base text-body-base text-on-surface-variant mb-6">
              Ensure total compliance with audit-ready ESG reporting and traceable generation data,
              provenance-tagged end to end.
            </p>
            <div className="space-y-4 pt-4 border-t border-outline-variant">
              <ul className="space-y-2 font-body-base text-body-base text-on-surface-variant">
                <li className="flex items-center gap-2"><span className="text-secondary">✓</span> Traceable Records</li>
                <li className="flex items-center gap-2"><span className="text-secondary">✓</span> Automated ESG Reporting</li>
              </ul>
            </div>
          </div>

          <div className="md:col-span-6 bg-surface-container border border-outline-variant rounded-lg p-8 md:p-12 hover:border-primary-container transition-colors flex flex-col md:flex-row gap-8 items-center">
            <div className="md:w-1/2 space-y-6">
              <div className="flex items-center gap-3">
                <Zap size={22} className="text-primary" />
                <h2 className="font-headline-md text-headline-md text-on-surface">Grid Operators</h2>
              </div>
              <p className="text-[18px] leading-relaxed text-on-surface-variant">
                Maintain grid stability with real-time telemetry on a configurable polling interval. Clean
                Energy Analytics provides the high-frequency data required for dynamic load balancing and
                predictive maintenance.
              </p>
            </div>
            <div className="md:w-1/2 w-full">
              <div className="bg-surface-dim border border-outline-variant rounded-lg overflow-hidden shadow-2xl">
                <div className="bg-surface border-b border-outline-variant px-4 py-2 flex gap-2 items-center">
                  <div className="w-2 h-2 rounded-full bg-outline-variant" />
                  <div className="w-2 h-2 rounded-full bg-outline-variant" />
                  <div className="w-2 h-2 rounded-full bg-outline-variant" />
                  <div className="ml-4 font-data-mono-sm text-data-mono-sm text-on-surface-variant">telemetry_stream.cea</div>
                </div>
                <div className="p-6 space-y-4">
                  <div className="flex justify-between items-end border-b border-outline-variant pb-2">
                    <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Every Reading</span>
                    <span className="font-data-display-lg text-[22px] text-primary">Provenance-Tagged</span>
                  </div>
                  <div className="flex justify-between font-data-mono-sm text-data-mono-sm">
                    <span className="text-on-surface-variant">Polling Interval</span>
                    <span className="text-primary">Configurable</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>
      </main>
      <LandingFooter />
    </div>
  );
}
