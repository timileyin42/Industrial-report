import { Outlet } from "react-router-dom";
import { Sidebar, MobileNav } from "./Sidebar";

export function AppLayout() {
  return (
    <div className="min-h-screen text-on-background">
      <Sidebar />
      <main className="md:ml-[272px] flex flex-col min-h-screen pb-16 md:pb-0">
        <Outlet />
      </main>
      <MobileNav />
    </div>
  );
}
