import { Link } from "react-router-dom";
import { Logo } from "../../components/brand/Logo";

const footerLinkClass = "text-on-surface-variant hover:text-primary transition-colors font-body-base text-body-base";
const columnHeadingClass = "font-label-caps text-label-caps text-on-surface uppercase mb-4";

const COLUMNS: { heading: string; links: { to: string; label: string }[] }[] = [
  {
    heading: "Product",
    links: [
      { to: "/features", label: "Features" },
      { to: "/solutions", label: "Solutions" },
      { to: "/sandbox", label: "Data Sandbox" },
    ],
  },
  {
    heading: "Company",
    links: [
      { to: "/company", label: "About Us" },
      { to: "/company#contact", label: "Request a Demo" },
    ],
  },
  {
    heading: "Legal & Contact",
    links: [
      { to: "/privacy", label: "Privacy Policy" },
      { to: "/terms", label: "Terms of Service" },
      { to: "/security", label: "Security" },
    ],
  },
];

export function LandingFooter() {
  return (
    <footer className="w-full bg-surface-container-lowest border-t border-outline-variant">
      <div className="max-w-7xl mx-auto px-grid-margin py-16">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-10">
          <div className="md:col-span-1">
            <Link to="/" className="inline-flex items-center opacity-80 hover:opacity-100 transition-opacity" aria-label="Back to home">
              <Logo />
            </Link>
            <p className="text-on-surface-variant text-sm font-data-mono-sm mt-4">
              © 2026 Clean Energy Analytics. Industrial Data Infrastructure.
            </p>
          </div>
          {COLUMNS.map((column) => (
            <div key={column.heading}>
              <p className={columnHeadingClass}>{column.heading}</p>
              <ul className="flex flex-col gap-3">
                {column.links.map((link) => (
                  <li key={link.label}>
                    <Link to={link.to} className={footerLinkClass}>{link.label}</Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </div>
    </footer>
  );
}
