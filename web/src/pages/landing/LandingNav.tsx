import { useState } from "react";
import { Link, NavLink } from "react-router-dom";
import { Menu, X } from "lucide-react";
import { Logo } from "../../components/brand/Logo";

const linkClass = ({ isActive }: { isActive: boolean }) =>
  [
    "font-body-base text-body-base transition-colors duration-200",
    isActive ? "text-primary font-bold border-b-2 border-primary pb-1" : "text-on-surface-variant hover:text-primary",
  ].join(" ");

const mobileLinkClass = ({ isActive }: { isActive: boolean }) =>
  [
    "font-body-base text-body-lg py-3 border-b border-outline-variant transition-colors",
    isActive ? "text-primary font-bold" : "text-on-surface-variant",
  ].join(" ");

// One consistent nav across all 4 marketing pages — the Stitch export had
// two slightly different nav treatments (logo+wordmark on Home, text-only
// wordmark on Features/Solutions/Company); unified here rather than
// carrying that inconsistency into the built site.
export function LandingNav() {
  const [menuOpen, setMenuOpen] = useState(false);

  return (
    <nav className="fixed top-0 w-full z-50 bg-surface-container-lowest/90 backdrop-blur-md border-b border-outline-variant">
      <div className="max-w-7xl mx-auto px-grid-margin flex justify-between items-center h-20">
        <Link to="/" className="flex items-center" onClick={() => setMenuOpen(false)}>
          <Logo />
        </Link>
        <div className="hidden md:flex items-center gap-8">
          <NavLink to="/features" className={linkClass}>Features</NavLink>
          <NavLink to="/solutions" className={linkClass}>Solutions</NavLink>
          <NavLink to="/company" className={linkClass}>Company</NavLink>
        </div>
        <div className="hidden md:flex items-center gap-4">
          <Link
            to="/login"
            className="bg-primary-container hover:bg-inverse-primary text-on-primary-container font-semibold px-4 py-2 rounded transition-colors"
          >
            Sign In
          </Link>
        </div>
        <button
          className="md:hidden text-on-surface"
          aria-label={menuOpen ? "Close menu" : "Open menu"}
          onClick={() => setMenuOpen((v) => !v)}
        >
          {menuOpen ? <X size={26} /> : <Menu size={26} />}
        </button>
      </div>
      {menuOpen && (
        <div className="md:hidden bg-surface-container-lowest border-t border-outline-variant px-grid-margin flex flex-col">
          <NavLink to="/features" className={mobileLinkClass} onClick={() => setMenuOpen(false)}>Features</NavLink>
          <NavLink to="/solutions" className={mobileLinkClass} onClick={() => setMenuOpen(false)}>Solutions</NavLink>
          <NavLink to="/company" className={mobileLinkClass} onClick={() => setMenuOpen(false)}>Company</NavLink>
          <Link
            to="/login"
            className="my-4 text-center bg-primary-container text-on-primary-container font-semibold px-4 py-3 rounded transition-colors"
            onClick={() => setMenuOpen(false)}
          >
            Sign In
          </Link>
        </div>
      )}
    </nav>
  );
}
