import { createContext, useContext, useState, type ReactNode } from "react";

const STORAGE_KEY = "cea.sidebarCollapsed";

interface SidebarContextValue {
  collapsed: boolean;
  toggle: () => void;
}

const SidebarContext = createContext<SidebarContextValue | null>(null);

// Shared between Sidebar (renders icon-only when collapsed), AppLayout
// (adjusts the main content's left margin to match), and TopNav (hosts
// the toggle button) — a context rather than prop-drilling since TopNav
// is rendered per-page, not by AppLayout itself.
export function SidebarProvider({ children }: { children: ReactNode }) {
  const [collapsed, setCollapsed] = useState(() => {
    try {
      return localStorage.getItem(STORAGE_KEY) === "true";
    } catch {
      return false;
    }
  });

  function toggle() {
    setCollapsed((prev) => {
      const next = !prev;
      try {
        localStorage.setItem(STORAGE_KEY, String(next));
      } catch {
        // localStorage unavailable (private browsing, etc.) — collapse
        // state just won't persist across reloads, not worth failing over.
      }
      return next;
    });
  }

  return <SidebarContext.Provider value={{ collapsed, toggle }}>{children}</SidebarContext.Provider>;
}

export function useSidebar() {
  const ctx = useContext(SidebarContext);
  if (!ctx) throw new Error("useSidebar must be used within SidebarProvider");
  return ctx;
}
