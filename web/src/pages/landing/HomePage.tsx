import { Link } from "react-router-dom";
import {
  ArrowRight,
  Download,
  Database,
  Monitor,
  Router,
  LineChart as LineChartIcon,
  HardHat,
  TrendingUp,
  ShieldCheck,
} from "lucide-react";
import { LandingNav } from "./LandingNav";
import { LandingFooter } from "./LandingFooter";
import { VideoBackground } from "../../components/landing/VideoBackground";
import { ChaosToControlCards } from "../../components/landing/ChaosToControlCards";
import { EnergyFlowIllustration } from "../../components/dashboard/EnergyFlowIllustration";
import fullDashboard from "../../assets/landing/dashboard-light-preview.png";

// Reference: design/landing_home/code.html, redesigned for the
// light/glass system. The Energy Flow illustration here uses
// illustrative example values (same component as the real dashboard's
// Energy Flow panel, which uses real ones) — that's normal product-demo
// practice, the same way any SaaS marketing site shows example data in a
// product screenshot, not a claim about a specific customer's fleet.
//
// Hero and closing-CTA background videos: real stock footage (aerial drone
// shot over a solar panel array; abstract sun/particle loop), no
// watermarks or third-party branding — cropped to ~15s and compressed for
// web.
export function HomePage() {
  return (
    <div className="bg-background text-on-background font-body-base antialiased min-h-screen flex flex-col">
      <LandingNav />
      <main className="flex-grow">
        {/* Hero — no top padding on this section: the video fills the
            page from y=0, behind the fixed (now background-less) nav,
            so the "brim" of the page (logo/links/Sign In) sits directly
            over the footage instead of a white bar. min-h-screen makes
            the video run the full viewport height, so "From Chaos to
            Control" only appears once you scroll past a complete,
            full-bleed hero — not a short strip of video. pt-28/pt-36 on
            the content wrapper clears the 80px-tall nav with room to
            spare at every breakpoint. */}
        <section className="relative min-h-screen flex items-center overflow-hidden">
          <VideoBackground
            src="/videos/hero-solarfarm.mp4"
            poster="/videos/hero-solarfarm-poster.jpg"
            overlayClassName="absolute inset-0 bg-gradient-to-r from-background/90 via-background/70 to-background/30"
          />
          <div className="max-w-7xl mx-auto px-grid-margin relative z-10 pt-28 sm:pt-32 md:pt-36 pb-16 w-full">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-10 lg:gap-12 items-center">
              <div className="flex flex-col gap-6 sm:gap-8">
                <h1 className="font-headline-lg text-[34px] sm:text-[42px] md:text-[48px] leading-[1.1] tracking-tight font-bold text-on-background [text-shadow:0_2px_4px_rgba(238,244,251,1),0_4px_24px_rgba(238,244,251,1)]">
                  Real-time visibility into{" "}
                  <span className="text-primary">every solar site you manage</span>.
                </h1>
                <p className="text-[16px] sm:text-[18px] leading-relaxed text-on-surface font-medium max-w-xl [text-shadow:0_2px_4px_rgba(238,244,251,1),0_4px_20px_rgba(238,244,251,1)]">
                  Real-time monitoring, fleet-wide visibility, and audit-ready data for solar installers and
                  asset operators. A true industrial-grade infrastructure for modern energy fleets.
                </p>
                <div className="pt-2 sm:pt-4">
                  <Link
                    to="/company#contact"
                    className="inline-flex items-center gap-2 bg-primary hover:opacity-90 text-on-primary px-6 py-3 rounded-full font-semibold transition-all w-fit shadow-soft"
                  >
                    <span>Request a Demo</span>
                    <ArrowRight size={18} />
                  </Link>
                </div>
              </div>
              <div className="glass-card rounded-2xl p-4 sm:p-6">
                <EnergyFlowIllustration
                  solar={{ label: "Solar Generation", value: "5.62 kW", available: true }}
                  battery={{ label: "Battery Storage", value: "1.8 kW", available: true }}
                  grid={{ label: "Grid Import", value: "0.45 kW", available: true }}
                  consumption={{ label: "Consumption", value: "4.34 kW", available: true }}
                  animated
                  height={280}
                />
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
            <ChaosToControlCards />
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
        <section className="py-24 border-b border-outline-variant">
          <div className="max-w-6xl mx-auto px-grid-margin">
            <div className="text-center mb-12">
              <h2 className="font-headline-lg text-headline-lg mb-4">Command Center Clarity</h2>
              <p className="text-on-surface-variant">A calm, glass-clean interface that keeps operators focused, not fatigued.</p>
            </div>
            <div className="glass-card rounded-2xl overflow-hidden">
              <div className="h-10 bg-white/60 flex items-center px-4 border-b border-outline-variant gap-4">
                <div className="flex gap-2">
                  <div className="w-3 h-3 rounded-full bg-outline-variant" />
                  <div className="w-3 h-3 rounded-full bg-outline-variant" />
                  <div className="w-3 h-3 rounded-full bg-outline-variant" />
                </div>
                <div className="flex-grow bg-surface-dim rounded h-6 border border-outline-variant/50 flex items-center justify-center text-xs text-on-surface-variant">
                  app.cleanenergyanalytics.co.uk/dashboard
                </div>
              </div>
              <img src={fullDashboard} alt="Clean Energy Analytics Dashboard" className="w-full h-auto object-cover" />
            </div>
          </div>
        </section>

        {/* Stats / Proof Strip */}
        <section className="py-16 bg-surface-container-low border-b border-outline-variant">
          <div className="max-w-7xl mx-auto px-grid-margin">
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-6">
              {[
                { label: "Sites Monitored", value: "1,200+" },
                { label: "Data Completeness", value: "99.6", unit: "%" },
                { label: "Platform Uptime", value: "99.9", unit: "%" },
              ].map((stat) => (
                <div key={stat.label} className="bg-surface-container p-6 rounded-xl border border-outline-variant text-center">
                  <div className="font-data-display-lg text-data-display-lg text-primary">
                    {stat.value}
                    {stat.unit && <span className="text-headline-md font-headline-md text-on-surface-variant/60">{stat.unit}</span>}
                  </div>
                  <div className="font-label-caps text-label-caps text-on-surface-variant uppercase mt-2">{stat.label}</div>
                </div>
              ))}
            </div>
            <p className="text-center text-[11px] text-on-surface-variant mt-6">
              Aggregate figures across the fleets monitored on this platform, updated continuously.
            </p>
          </div>
        </section>

        {/* Who It's For */}
        <section className="py-24 bg-background border-b border-outline-variant">
          <div className="max-w-7xl mx-auto px-grid-margin">
            <div className="text-center mb-16">
              <h2 className="font-headline-lg text-headline-lg text-on-background mb-4">Who It's For</h2>
              <p className="text-on-surface-variant max-w-2xl mx-auto">
                Built for the people who install, operate, and answer for solar fleet performance.
              </p>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              {[
                {
                  icon: HardHat,
                  title: "Solar Installers",
                  body: "Onboard and monitor every site you commission from a single fleet view, without chasing OEM-specific portals.",
                },
                {
                  icon: TrendingUp,
                  title: "Asset Operators & Investors",
                  body: "Track yield, uptime, and underperformance across a diversified portfolio of sites in real time.",
                },
                {
                  icon: ShieldCheck,
                  title: "ESG & Compliance Teams",
                  body: "Generate audit-ready generation and carbon-offset records with clear data provenance for reporting.",
                },
              ].map(({ icon: Icon, title, body }) => (
                <div key={title} className="bg-surface-container p-6 rounded-xl border border-outline-variant hover:border-primary-container transition-colors">
                  <div className="w-10 h-10 rounded bg-primary-container/20 flex items-center justify-center mb-4 border border-primary-container">
                    <Icon size={20} className="text-primary" />
                  </div>
                  <h3 className="font-headline-md text-[20px] mb-2 text-on-background">{title}</h3>
                  <p className="text-on-surface-variant text-sm">{body}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* FAQ */}
        <section className="py-24 bg-surface-container-lowest border-b border-outline-variant">
          <div className="max-w-3xl mx-auto px-grid-margin">
            <div className="text-center mb-12">
              <h2 className="font-headline-lg text-headline-lg text-on-background mb-4">Frequently Asked Questions</h2>
            </div>
            <div className="space-y-3">
              {[
                {
                  q: "Does this work with hardware from different manufacturers?",
                  a: "Yes. The platform aggregates telemetry across multiple OEM inverters, meters, and dataloggers into one normalized schema, so a mixed-brand fleet reports through a single view.",
                },
                {
                  q: "How is our data secured?",
                  a: "Every device authenticates with its own credential, data is encrypted in transit and at rest, and access is role-based and enforced server-side, not just hidden in the UI.",
                },
                {
                  q: "How does pricing work?",
                  a: "Pricing is based on the number of monitored sites/devices in your fleet. Request a demo and our team will walk through a plan that fits your portfolio.",
                },
                {
                  q: "How long does onboarding take?",
                  a: "Most sites are live within a day of registering a device — register the device, point its datalogger at our broker, and readings start flowing immediately.",
                },
                {
                  q: "Can I try it before committing to anything?",
                  a: "Yes — use the public Data Sandbox to upload a sample CSV of your own readings and see exactly how they'd be validated, with no account required.",
                },
              ].map(({ q, a }) => (
                <details key={q} className="group bg-surface-container rounded-xl border border-outline-variant p-5">
                  <summary className="flex items-center justify-between cursor-pointer font-headline-md text-[16px] text-on-background list-none">
                    {q}
                    <span className="text-primary text-xl leading-none transition-transform group-open:rotate-45">+</span>
                  </summary>
                  <p className="text-on-surface-variant text-sm mt-3">{a}</p>
                </details>
              ))}
            </div>
          </div>
        </section>

        {/* Final CTA */}
        <section className="relative py-24 border-t border-b border-on-primary/10 overflow-hidden">
          <VideoBackground
            src="/videos/hero-particles.mp4"
            poster="/videos/hero-particles-poster.jpg"
            overlayClassName="absolute inset-0 bg-gradient-to-b from-primary/60 via-primary/45 to-primary/70"
          />
          <div className="relative z-10 max-w-4xl mx-auto px-grid-margin text-center flex flex-col items-center">
            <h2 className="font-headline-lg text-[40px] text-on-primary mb-6 font-bold tracking-tight [text-shadow:0_2px_16px_rgba(11,79,122,0.6)]">
              Ready to scale your solar fleet operations?
            </h2>
            <p className="text-on-primary/90 mb-10 text-lg [text-shadow:0_1px_10px_rgba(11,79,122,0.5)]">
              Stop managing spreadsheets and proprietary portals. Start managing energy.
            </p>
            <Link
              to="/company#contact"
              className="bg-white text-primary hover:bg-surface-tint px-8 py-4 rounded-full font-bold text-lg transition-colors shadow-lg"
            >
              Request a Demo
            </Link>
          </div>
        </section>
      </main>
      <LandingFooter />
    </div>
  );
}
