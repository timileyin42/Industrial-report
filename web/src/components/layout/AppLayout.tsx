import { Outlet } from "react-router-dom";
import { Sidebar, MobileNav } from "./Sidebar";
import { SidebarProvider, useSidebar } from "./SidebarContext";

function AppLayoutContent() {
  const { collapsed } = useSidebar();
  return (
    <div className="min-h-screen text-on-background">
      <Sidebar />
      <main className={`${collapsed ? "md:ml-[104px]" : "md:ml-[272px]"} flex flex-col min-h-screen pb-16 md:pb-0 transition-[margin] duration-200`}>
        <Outlet />
      </main>
      <MobileNav />
    </div>
  );
}

export function AppLayout() {
  return (
    <SidebarProvider>
      <AppLayoutContent />
    </SidebarProvider>
  );
}
