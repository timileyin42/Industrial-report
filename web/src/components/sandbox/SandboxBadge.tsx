import { FlaskConical } from "lucide-react";

// Fixed top-left, always red, always on top — the whole point is that
// nobody looking at this page could ever mistake it for the real app or
// real fleet data, even mid-scroll.
export function SandboxBadge() {
  return (
    <div className="fixed top-3 left-3 z-[100] flex items-center gap-1.5 bg-error text-white text-[11px] font-bold uppercase tracking-wide px-3 py-1.5 rounded-full shadow-lg">
      <FlaskConical size={13} />
      Sandbox — Testing
    </div>
  );
}
