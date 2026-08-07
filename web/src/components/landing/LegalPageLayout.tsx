import type { ReactNode } from "react";
import { LandingNav } from "../../pages/landing/LandingNav";
import { LandingFooter } from "../../pages/landing/LandingFooter";

// Shared shell for the three statutory/policy pages (Terms, Privacy,
// Security) — same nav/footer/prose treatment as the other marketing
// pages, just no video hero since these need to read as documents, not
// a pitch.
export function LegalPageLayout({
  title,
  lastUpdated,
  children,
}: {
  title: string;
  lastUpdated: string;
  children: ReactNode;
}) {
  return (
    <div className="bg-background text-on-background font-body-base antialiased min-h-screen flex flex-col">
      <LandingNav />
      <main className="flex-grow pt-32 pb-24">
        <div className="max-w-3xl mx-auto px-grid-margin">
          <h1 className="font-headline-lg text-headline-lg text-on-background mb-2">{title}</h1>
          <p className="text-on-surface-variant text-sm mb-10">Last updated: {lastUpdated}</p>
          <div className="glass-card rounded-2xl p-8 sm:p-10 space-y-8 prose-legal">{children}</div>
        </div>
      </main>
      <LandingFooter />
    </div>
  );
}

// Shared prose primitives so every section across the three pages reads
// consistently without repeating className strings everywhere.
export function LegalSection({ heading, children }: { heading: string; children: ReactNode }) {
  return (
    <section className="space-y-3">
      <h2 className="font-headline-md text-headline-md text-on-surface">{heading}</h2>
      <div className="text-on-surface-variant text-[15px] leading-relaxed space-y-3">{children}</div>
    </section>
  );
}
