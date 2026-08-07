import { Link } from "react-router-dom";
import { Logo } from "../../components/brand/Logo";

const footerLinkClass = "text-on-surface-variant hover:text-primary transition-colors font-body-base text-body-base";

export function LandingFooter() {
  return (
    <footer className="w-full bg-surface-container-lowest border-t border-outline-variant py-grid-margin">
      <div className="max-w-7xl mx-auto px-grid-margin flex flex-col md:flex-row justify-between items-center gap-gutter">
        <div className="flex items-center gap-4">
          <Link to="/" className="opacity-80 hover:opacity-100 transition-opacity" aria-label="Back to home">
            <Logo />
          </Link>
          <span className="text-on-surface-variant text-sm font-data-mono-sm">
            © 2026 Clean Energy Analytics. Industrial Data Infrastructure.
          </span>
        </div>
        <div className="flex gap-6">
          <Link to="/privacy" className={footerLinkClass}>Privacy Policy</Link>
          <Link to="/terms" className={footerLinkClass}>Terms of Service</Link>
          <Link to="/security" className={footerLinkClass}>Security</Link>
        </div>
      </div>
    </footer>
  );
}
