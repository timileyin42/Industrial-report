import { useEffect, useState } from "react";
import { Link, NavLink } from "react-router-dom";
import { Menu, X } from "lucide-react";
import { Logo } from "../../components/brand/Logo";

const linkClass = ({ isActive }: { isActive: boolean }) =>
  [
    "font-body-base text-body-base transition-colors duration-200 drop-shadow-sm",
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
//
// Transparent at the very top of the page on purpose: on Home it sits
// directly over the hero video (the whole point — video fills the page
// edge-to-edge, nav floats over it). But past the hero, this same fixed
// bar sits over ordinary scrolling content (text, cards, illustrations)
// with no background of its own — that content visibly shows through
// behind the logo/links, which reads as broken rather than intentional.
// landing-nav-scrolled (index.css) kicks in once you scroll past a small
// threshold, giving the bar a real background so it stops overlapping
// whatever's behind it.
export function LandingNav() {
  const [menuOpen, setMenuOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    function onScroll() {
      setScrolled(window.scrollY > 24);
    }
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  return (
    <nav className={`fixed top-0 w-full z-50 transition-[background-color,box-shadow] duration-200 ${scrolled ? "landing-nav-scrolled" : ""}`}>
      <div className="max-w-7xl mx-auto px-grid-margin flex justify-between items-center h-20">
        <Link to="/" className="flex items-center drop-shadow-sm" onClick={() => setMenuOpen(false)}>
          <Logo />
        </Link>
        <div className="hidden md:flex items-center gap-8">
          <NavLink to="/features" className={linkClass}>Features</NavLink>
          <NavLink to="/solutions" className={linkClass}>Solutions</NavLink>
          <NavLink to="/company" className={linkClass}>Company</NavLink>
          <NavLink to="/sandbox" className={linkClass}>Sandbox</NavLink>
        </div>
        <div className="hidden md:flex items-center gap-4">
          <Link
            to="/login"
            className="bg-primary-container hover:bg-inverse-primary text-on-primary-container font-semibold px-4 py-2 rounded shadow-soft transition-colors"
          >
            Sign In
          </Link>
        </div>
        <button
          className="md:hidden text-on-surface drop-shadow-sm"
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
          <NavLink to="/sandbox" className={mobileLinkClass} onClick={() => setMenuOpen(false)}>Sandbox</NavLink>
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
