import { Link } from "react-router-dom";
import { ArrowRight, Download, Database, Monitor, Router, LineChart as LineChartIcon } from "lucide-react";
import { LandingNav } from "./LandingNav";
import { LandingFooter } from "./LandingFooter";
import { NetworkHeroBackground } from "../../components/landing/NetworkHeroBackground";
import fullDashboard from "../../assets/landing/site-power-chart.png";

// Reference: design/landing_home/code.html. The hero no longer carries
// the mockup's "Uptime SLA 99.99%" / "Data Resolution 1s" stats or the
// "Fleet-Scale Proof Strip" section ("5,000+ Sites Monitored", "1.2M
// Tons CO2 Avoided", etc.) — all fabricated, none backed by anything
// this platform can actually measure. The hero's "Request a Demo" / "See
// it in action" buttons are gone too — neither went anywhere (no demo
// booking flow, no product video exists); the one CTA that remains
// (Sign In) is the one thing on this page that's actually real.
export function HomePage() {
  return (
    <div className="bg-background text-on-background font-body-base antialiased min-h-screen flex flex-col">
      <LandingNav />
      <main className="flex-grow pt-[80px]">
        {/* Hero */}
        <section className="relative pt-24 pb-32 overflow-hidden border-b border-outline-variant">
          <NetworkHeroBackground />
          <div className="max-w-7xl mx-auto px-grid-margin relative z-10">
            <div className="max-w-2xl flex flex-col gap-8">
              <h1 className="font-headline-lg text-[48px] leading-[1.1] tracking-tight font-bold text-on-background">
                Turning scattered solar data into trustworthy,{" "}
                <span className="text-primary">verification-ready insight</span> at fleet scale.
              </h1>
              <p className="text-[18px] leading-relaxed text-on-surface-variant max-w-xl">
                Real-time monitoring, fleet-wide visibility, and audit-ready data for solar installers and
                asset operators. A true industrial-grade infrastructure for modern energy fleets.
              </p>
              <div className="pt-4">
                <Link
                  to="/login"
                  className="inline-flex items-center gap-2 bg-primary-container hover:bg-inverse-primary text-on-primary-container px-6 py-3 rounded font-semibold transition-all w-fit"
                >
                  <span>Sign In</span>
                  <ArrowRight size={18} />
                </Link>
              </div>
            </div>
          </div>
        </section>

        {/* Problem / Value */}
        <section className="py-24 bg-surface-container-lowest border-b border-outline-variant">
          <div className="max-w-7xl mx-auto px-grid-margin">
            <div className="text-center mb-16">
              <h2 className="font-headline-lg text-headline-lg text-on-background mb-4">From Chaos to Control</h2>
              <p className="text-on-surface-variant max-w-2xl mx-auto">
                Standardize disparate telemetry into a single, unified operational source of truth.
              </p>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
              {[
                { from: "Fragmented Data", to: "Unified Fleet View", body: "Aggregate inverter, meter, and weather station data across multiple OEMs into one normalized schema." },
                { from: "Opaque Status", to: "Verified Reporting", body: "Real-time device-level status monitoring with data provenance for full auditability." },
                { from: "Unstructured Data", to: "Audit-Ready ESG", body: "Generate structured reports for carbon avoidance and energy yield tailored for regulatory compliance." },
              ].map((item) => (
                <div key={item.from} className="bg-surface-container-low p-6 rounded-lg border border-outline-variant flex flex-col gap-4">
                  <div className="flex justify-between items-center pb-4 border-b border-outline-variant">
                    <span className="text-error/80 font-semibold line-through">{item.from}</span>
                    <span className="text-outline">→</span>
                    <span className="text-primary font-bold">{item.to}</span>
                  </div>
                  <p className="text-on-surface-variant text-sm mt-2">{item.body}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* Core Capabilities */}
        <section className="py-24 bg-background border-b border-outline-variant">
          <div className="max-w-7xl mx-auto px-grid-margin">
            <h2 className="font-headline-lg text-headline-lg mb-12">Core Platform Capabilities</h2>
            <div className="grid grid-cols-1 md:grid-cols-6 gap-6">
              {[
                { icon: Download, title: "Ingest", body: "Multi-brand protocol aggregation (Modbus, DNP3, OPC UA) at the edge." },
                { icon: Database, title: "Store", body: "High-availability time-series storage built for millions of concurrent telemetry streams." },
                { icon: Monitor, title: "Visualize", body: "Real-time performance dashboards tailored for NOCs and control rooms." },
              ].map(({ icon: Icon, title, body }) => (
                <div key={title} className="col-span-1 md:col-span-2 bg-surface-container p-6 rounded-xl border border-outline-variant hover:border-primary-container transition-colors">
                  <div className="w-10 h-10 rounded bg-primary-container/20 flex items-center justify-center mb-4 border border-primary-container">
                    <Icon size={20} className="text-primary" />
                  </div>
                  <h3 className="font-headline-md text-[20px] mb-2 text-on-background">{title}</h3>
                  <p className="text-on-surface-variant text-sm">{body}</p>
                </div>
              ))}
              <div className="col-span-1 md:col-span-3 bg-surface-container p-6 rounded-xl border border-outline-variant hover:border-primary-container transition-colors flex items-start gap-6">
                <div className="flex-shrink-0 w-12 h-12 rounded bg-primary-container/20 flex items-center justify-center border border-primary-container mt-1">
                  <Router size={24} className="text-primary" />
                </div>
                <div>
                  <h3 className="font-headline-md text-[20px] mb-2 text-on-background">Manage Devices</h3>
                  <p className="text-on-surface-variant text-sm max-w-sm mb-4">
                    Remote telemetry and status monitoring across your entire hardware fleet.
                  </p>
                  <div className="flex gap-2">
                    <span className="px-2 py-1 bg-surface-variant rounded text-xs text-on-surface font-data-mono-sm">OTA Updates</span>
                    <span className="px-2 py-1 bg-surface-variant rounded text-xs text-on-surface font-data-mono-sm">Remote Reboot</span>
                  </div>
                </div>
              </div>
              <div className="col-span-1 md:col-span-3 bg-surface-container p-6 rounded-xl border border-outline-variant hover:border-primary-container transition-colors flex items-start gap-6">
                <div className="flex-shrink-0 w-12 h-12 rounded bg-primary-container/20 flex items-center justify-center border border-primary-container mt-1">
                  <LineChartIcon size={24} className="text-primary" />
                </div>
                <div>
                  <h3 className="font-headline-md text-[20px] mb-2 text-on-background">Analyze</h3>
                  <p className="text-on-surface-variant text-sm max-w-sm">
                    Automated yield and efficiency insights. Detect underperformance anomalies before they impact returns.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Product Showcase */}
        <section className="py-24 bg-surface-container-lowest border-b border-outline-variant">
          <div className="max-w-6xl mx-auto px-grid-margin">
            <div className="text-center mb-12">
              <h2 className="font-headline-lg text-headline-lg mb-4">Command Center Fidelity</h2>
              <p className="text-on-surface-variant">Built for the control room. Dark-mode native to reduce operator fatigue.</p>
            </div>
            <div className="rounded-xl border border-outline-variant bg-surface-container-high shadow-2xl overflow-hidden">
              <div className="h-10 bg-surface flex items-center px-4 border-b border-outline-variant gap-4">
                <div className="flex gap-2">
                  <div className="w-3 h-3 rounded-full bg-outline-variant" />
                  <div className="w-3 h-3 rounded-full bg-outline-variant" />
                  <div className="w-3 h-3 rounded-full bg-outline-variant" />
                </div>
                <div className="flex-grow bg-surface-container-low rounded h-6 border border-outline-variant/50 flex items-center justify-center text-xs text-on-surface-variant font-data-mono-sm">
                  app.cleanenergyanalytics.co.uk/dashboard
                </div>
              </div>
              <div className="bg-background relative">
                <img src={fullDashboard} alt="Full Clean Energy Analytics Dashboard" className="w-full h-auto object-cover opacity-90" />
                <div className="absolute inset-0 bg-gradient-to-t from-background via-transparent to-transparent opacity-60" />
              </div>
            </div>
          </div>
        </section>

        {/* Final CTA */}
        <section className="py-24 bg-primary-container border-t border-b border-on-primary-container/20">
          <div className="max-w-4xl mx-auto px-grid-margin text-center flex flex-col items-center">
            <h2 className="font-headline-lg text-[40px] text-white mb-6 font-bold tracking-tight">
              Ready to scale your solar fleet operations?
            </h2>
            <p className="text-primary-fixed mb-10 text-lg">
              Stop managing spreadsheets and proprietary portals. Start managing energy.
            </p>
            <Link
              to="/login"
              className="bg-white text-primary-container hover:bg-surface-tint px-8 py-4 rounded font-bold text-lg transition-colors shadow-lg"
            >
              Sign In
            </Link>
          </div>
        </section>
      </main>
      <LandingFooter />
    </div>
  );
}
