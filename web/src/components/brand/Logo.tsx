import logoFullSrc from "../../assets/brand/logo-full.svg";
import logoMarkSrc from "../../assets/brand/logo-mark.svg";

// Real brand assets (web/src/assets/brand/) — logo-full.svg is the
// provided lockup with its baked-in background rect stripped out (the
// source file's own comment invited this: "remove this rect if you want
// a transparent logo") so it drops cleanly onto whatever surface token
// it's placed on, rather than carrying a fixed navy box that wouldn't
// exactly match any of them. logo-mark.svg is that same file's bar-chart
// glyph cropped out on its own, for spots too small for the full
// two-line wordmark (favicon, the small icon next to nav/sidebar text).
export function LogoMark({ size = 22 }: { size?: number }) {
  return <img src={logoMarkSrc} alt="" aria-hidden="true" style={{ height: size, width: "auto" }} />;
}

export function Logo() {
  return <img src={logoFullSrc} alt="Clean Energy Analytics" className="h-9 w-auto" />;
}
