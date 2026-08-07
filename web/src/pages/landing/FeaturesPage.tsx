import { Share2, Database, BarChart3, RadioTower, TrendingUp, Monitor } from "lucide-react";
import { LandingNav } from "./LandingNav";
import { LandingFooter } from "./LandingFooter";
import performanceViz from "../../assets/landing/dashboard-light-preview.png";

// Reference: design/landing_features/code.html.
export function FeaturesPage() {
  return (
    <div className="bg-background text-on-background min-h-screen flex flex-col antialiased">
      <LandingNav />
      <main className="flex-grow pt-32 pb-24 max-w-7xl mx-auto w-full px-grid-margin">
        <header className="mb-16 md:mb-24 md:max-w-3xl">
          <h1 className="font-headline-lg text-headline-lg text-on-surface mb-6">Core Capabilities</h1>
          <p className="text-[18px] leading-relaxed text-on-surface-variant">
            The industrial data infrastructure built for utility-scale solar fleet operations. Aggregate,
            store, and act on telemetry at the edge and in the cloud with absolute control.
          </p>
        </header>

        <div className="grid grid-cols-1 md:grid-cols-6 gap-6">
          <div className="md:col-span-4 bg-surface-container border border-outline-variant rounded-lg p-8 hover:border-primary-container transition-colors">
            <div className="flex items-center gap-3 mb-4">
              <Share2 size={28} className="text-primary" />
              <h2 className="font-headline-md text-headline-md text-on-surface">Ingest</h2>
            </div>
            <p className="font-body-base text-body-base text-on-surface-variant mb-6 max-w-xl">
              Multi-protocol edge aggregation capable of handling high-frequency telemetry from diverse OEM
              inverters, trackers, and weather stations simultaneously.
            </p>
            <div className="bg-surface-dim border border-outline-variant rounded p-4 font-data-mono-sm text-data-mono-sm text-on-surface-variant">
              <div className="flex justify-between border-b border-outline-variant pb-2 mb-2">
                <span>Protocol Support</span>
                <span className="text-primary">Native</span>
              </div>
              <div className="grid grid-cols-2 gap-x-4 gap-y-2">
                <div>&gt; Modbus TCP/RTU</div>
                <div>&gt; DNP3</div>
                <div>&gt; OPC UA</div>
                <div>&gt; MQTT</div>
              </div>
            </div>
          </div>

          <div className="md:col-span-2 bg-surface-container border border-outline-variant rounded-lg p-8 hover:border-primary-container transition-colors flex flex-col">
            <div className="flex items-center gap-3 mb-4">
              <Database size={28} className="text-primary" />
              <h2 className="font-headline-md text-headline-md text-on-surface">Store</h2>
            </div>
            <p className="font-body-base text-body-base text-on-surface-variant mb-6">
              Purpose-built time-series storage designed for massive horizontal scalability across
              distributed sites.
            </p>
            <div className="mt-auto">
              <div className="flex items-end gap-2 mb-2">
                <span className="font-data-display-lg text-data-display-lg text-primary">10x</span>
                <span className="font-label-caps text-label-caps text-on-surface-variant pb-1">COMPRESSION</span>
              </div>
              <div className="w-full bg-surface-dim h-1 rounded overflow-hidden">
                <div className="bg-primary h-full w-4/5" />
              </div>
            </div>
          </div>

          <div className="md:col-span-3 bg-surface-container border border-outline-variant rounded-lg p-8 hover:border-primary-container transition-colors border-l-2 border-l-primary">
            <div className="flex justify-between items-start mb-4">
              <div className="flex items-center gap-3">
                <BarChart3 size={28} className="text-primary" />
                <h2 className="font-headline-md text-headline-md text-on-surface">Visualize</h2>
              </div>
              <span className="bg-surface-variant px-2 py-1 rounded font-data-mono-sm text-data-mono-sm text-on-surface-variant border border-outline-variant">
                UI Canvas
              </span>
            </div>
            <p className="font-body-base text-body-base text-on-surface-variant mb-6">
              High-fidelity performance dashboards providing real-time operational awareness. Command
              center grade visualization of asset health and grid compliance.
            </p>
            <div
              className="h-40 border border-outline-variant rounded relative overflow-hidden flex items-center justify-center bg-cover bg-center"
              style={{ backgroundImage: `url(${performanceViz})` }}
            >
              <div className="absolute inset-0 bg-surface-dim/40 backdrop-blur-[2px]" />
              <Monitor size={40} className="text-on-surface-variant relative z-10 opacity-50" />
            </div>
          </div>

          <div className="md:col-span-3 grid grid-rows-2 gap-6">
            <div className="bg-surface-container border border-outline-variant rounded-lg p-6 hover:border-primary-container transition-colors flex flex-col justify-center">
              <div className="flex items-center gap-3 mb-3">
                <RadioTower size={28} className="text-secondary" />
                <h2 className="font-headline-md text-headline-md text-on-surface">Manage</h2>
              </div>
              <p className="font-body-base text-body-base text-on-surface-variant mb-4">
                Secure, remote fleet management. Execute coordinated firmware deployments and command
                dispatches across thousands of edge nodes.
              </p>
              <div className="flex gap-2">
                <span className="bg-surface-variant px-2 py-1 rounded font-data-mono-sm text-data-mono-sm text-primary border border-outline-variant">&gt; OTA Updates</span>
                <span className="bg-surface-variant px-2 py-1 rounded font-data-mono-sm text-data-mono-sm text-on-surface-variant border border-outline-variant">Audit Logs</span>
              </div>
            </div>
            <div className="bg-surface-container border border-outline-variant rounded-lg p-6 hover:border-primary-container transition-colors flex flex-col justify-center">
              <div className="flex items-center gap-3 mb-3">
                <TrendingUp size={28} className="text-primary" />
                <h2 className="font-headline-md text-headline-md text-on-surface">Analyze</h2>
              </div>
              <p className="font-body-base text-body-base text-on-surface-variant">
                Compute yield metrics and efficiency analytics. Built-in anomaly detection isolates
                underperforming assets before they impact revenue.
              </p>
            </div>
          </div>
        </div>
      </main>
      <LandingFooter />
    </div>
  );
}
