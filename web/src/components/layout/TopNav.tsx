import { UserCircle } from "lucide-react";
import { useAuth } from "../../auth/AuthContext";

export function TopNav({ title }: { title: string }) {
  const { session } = useAuth();

  return (
    <header className="h-20 flex items-center justify-between px-grid-margin sticky top-0 z-40 bg-background/85 backdrop-blur-md">
      <h2 className="font-headline-md text-headline-md font-bold text-on-surface">{title}</h2>
      <div className="glass-card rounded-full flex items-center gap-3 pl-4 pr-2 py-2">
        <div className="text-right">
          <p className="font-label-caps text-label-caps text-on-surface leading-none">
            {session?.role === "operator" ? "Operator" : `Site: ${session?.siteId ?? ""}`}
          </p>
          <p className="text-[10px] text-primary uppercase font-bold mt-1">
            {session?.role === "operator" ? "Full Access" : "Restricted"}
          </p>
        </div>
        <UserCircle size={32} className="text-on-surface-variant" />
      </div>
    </header>
  );
}
