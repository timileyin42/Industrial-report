import { Link } from "react-router-dom";
import { ArrowRight, Download, Database, Monitor, Router, LineChart as LineChartIcon } from "lucide-react";
import { LandingNav } from "./LandingNav";
import { LandingFooter } from "./LandingFooter";
import { VideoBackground } from "../../components/landing/VideoBackground";
import { ChaosToControlCards } from "../../components/landing/ChaosToControlCards";
import { EnergyFlowIllustration } from "../../components/dashboard/EnergyFlowIllustration";
import fullDashboard from "../../assets/landing/dashboard-light-preview.png";

// Reference: design/landing_home/code.html, redesigned for the
// light/glass system. The hero no longer carries the mockup's fabricated
// "Uptime SLA 99.99%" stats or "Fleet-Scale Proof Strip" ("5,000+ Sites
// Monitored" etc.) — none backed by anything this platform can actually
// measure. The hero's dead "Request a Demo"/"See it in action" buttons
// are gone too; the one CTA that remains (Sign In) is the one thing on
// this page that's real. The Energy Flow illustration here uses
// illustrative example values (same component as the real dashboard's
// Energy Flow panel, which uses real ones) — that's normal product-demo
// practice, the same way any SaaS marketing site shows example data in a
// product screenshot, not a claim about a specific customer's fleet.
//
// Hero and closing-CTA background videos: real stock footage (aerial wind
// farm; abstract sun/particle loop), no watermarks or third-party
// branding — cropped to ~15s and compressed for web. Worth noting: the
// wind farm clip is wind energy, not solar, while this product's copy is
// solar-specific throughout. Used anyway for the visual (bright sky/green
// fields reads well against the light theme) — flagged here rather than
// silently glossed over.
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
            src="/videos/hero-windfarm.mp4"
            poster="/videos/hero-windfarm-poster.jpg"
            overlayClassName="absolute inset-0 bg-gradient-to-r from-background/65 via-background/35 to-background/10"
          />
          <div className="max-w-7xl mx-auto px-grid-margin relative z-10 pt-28 sm:pt-32 md:pt-36 pb-16 w-full">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-10 lg:gap-12 items-center">
              <div className="flex flex-col gap-6 sm:gap-8">
                <h1 className="font-headline-lg text-[34px] sm:text-[42px] md:text-[48px] leading-[1.1] tracking-tight font-bold text-on-background [text-shadow:0_2px_20px_rgba(238,244,251,0.9)]">
                  Real-time visibility into{" "}
                  <span className="text-primary">every solar site you manage</span>.
                </h1>
                <p className="text-[16px] sm:text-[18px] leading-relaxed text-on-surface-variant max-w-xl [text-shadow:0_2px_16px_rgba(238,244,251,0.95)]">
                  Real-time monitoring, fleet-wide visibility, and audit-ready data for solar installers and
                  asset operators. A true industrial-grade infrastructure for modern energy fleets.
                </p>
                <div className="pt-2 sm:pt-4">
                  <Link
                    to="/login"
                    className="inline-flex items-center gap-2 bg-primary hover:opacity-90 text-on-primary px-6 py-3 rounded-full font-semibold transition-all w-fit shadow-soft"
                  >
                    <span>Sign In</span>
                    <ArrowRight size={18} />
                  </Link>
                </div>
              </div>
              <div className="glass-card rounded-2xl p-4 sm:p-6">
                <EnergyFlowIllustration
                  solar={{ label: "Solar Generation", value: "5.62 kW", available: true }}
                  battery={{ label: "Battery Storage", value: "1.8 kW", available: true }}
                  grid={{ label: "Grid Import", value: "0.45 MW", available: true }}
                  consumption={{ label: "Consumption", value: "4.34 MW", available: true }}
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
              to="/login"
              className="bg-white text-primary hover:bg-surface-tint px-8 py-4 rounded-full font-bold text-lg transition-colors shadow-lg"
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
