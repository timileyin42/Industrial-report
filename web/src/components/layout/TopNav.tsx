import { HelpCircle } from "lucide-react";
import { Link } from "react-router-dom";
import { GlobalSearch } from "./GlobalSearch";
import { NotificationBell } from "./NotificationBell";
import { ProfileMenu } from "./ProfileMenu";

export function TopNav({ title }: { title: string }) {
  return (
    <header className="h-20 flex items-center justify-between gap-4 px-grid-margin sticky top-0 z-40 bg-background/85 backdrop-blur-md">
      <h2 className="font-headline-md text-headline-md font-bold text-on-surface flex-shrink-0">{title}</h2>
      <div className="hidden md:block flex-1 max-w-md">
        <GlobalSearch />
      </div>
      <div className="flex items-center gap-3 flex-shrink-0">
        <NotificationBell />
        <Link to="/app/help" className="glass-card rounded-full p-2.5 text-on-surface-variant hover:text-primary transition-colors" title="Help">
          <HelpCircle size={18} />
        </Link>
        <ProfileMenu />
      </div>
    </header>
  );
}
