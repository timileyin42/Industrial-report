import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { Search } from "lucide-react";
import { listSites } from "../../api/sites";
import { listDevices } from "../../api/devices";
import { useAuth } from "../../auth/AuthContext";

// Real search over already-fetched sites/devices (up to 200 of each) —
// no backend full-text search endpoint exists, so this is honest about
// its own scope rather than pretending to be exhaustive: a fleet beyond
// ~200 sites or devices needs a real search endpoint, not this. Matches
// on site_id/name/address and device_id, navigates on click.
export function GlobalSearch() {
  const navigate = useNavigate();
  const { session } = useAuth();
  const isOperator = session?.role === "operator";
  const [query, setQuery] = useState("");
  const [focused, setFocused] = useState(false);

  const sitesQuery = useQuery({
    queryKey: ["search-sites"],
    queryFn: () => listSites(undefined, 200),
    enabled: isOperator,
    staleTime: 60_000,
  });
  const devicesQuery = useQuery({
    queryKey: ["search-devices"],
    queryFn: () => listDevices({ limit: 200 }),
    enabled: isOperator,
    staleTime: 60_000,
  });

  const results = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return { sites: [], devices: [] };
    const sites = (sitesQuery.data?.items ?? [])
      .filter((s) => s.site_id.toLowerCase().includes(q) || s.name?.toLowerCase().includes(q) || s.address?.toLowerCase().includes(q))
      .slice(0, 5);
    const devices = (devicesQuery.data?.items ?? []).filter((d) => d.device_id.toLowerCase().includes(q)).slice(0, 5);
    return { sites, devices };
  }, [query, sitesQuery.data, devicesQuery.data]);

  if (!isOperator) return null;

  const hasResults = results.sites.length > 0 || results.devices.length > 0;

  return (
    <div className="relative w-full max-w-md">
      <div className="relative">
        <Search size={16} className="absolute left-4 top-1/2 -translate-y-1/2 text-on-surface-variant" />
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onFocus={() => setFocused(true)}
          onBlur={() => setTimeout(() => setFocused(false), 150)}
          placeholder="Search site, device…"
          className="w-full bg-white/70 border border-outline-variant rounded-full pl-10 pr-4 py-2 text-[14px] text-on-surface focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none placeholder:text-on-surface-variant/60"
        />
      </div>
      {focused && query.trim() && (
        <div className="absolute top-full mt-2 w-full glass-card rounded-xl overflow-hidden z-50">
          {!hasResults ? (
            <p className="px-4 py-3 text-[13px] text-on-surface-variant">No matches for "{query}"</p>
          ) : (
            <>
              {results.sites.map((s) => (
                <button
                  key={s.site_id}
                  onClick={() => {
                    navigate(`/app/sites/${s.site_id}`);
                    setQuery("");
                  }}
                  className="w-full text-left px-4 py-2.5 hover:bg-white/60 transition-colors flex items-center justify-between"
                >
                  <span className="text-on-surface text-[13px]">{s.name ?? s.site_id}</span>
                  <span className="text-on-surface-variant text-[11px] font-data-mono-sm">{s.site_id}</span>
                </button>
              ))}
              {results.devices.map((d) => (
                <button
                  key={d.device_id}
                  onClick={() => {
                    navigate("/app/devices");
                    setQuery("");
                  }}
                  className="w-full text-left px-4 py-2.5 hover:bg-white/60 transition-colors flex items-center justify-between"
                >
                  <span className="text-on-surface text-[13px] font-data-mono-sm">{d.device_id}</span>
                  <span className="text-on-surface-variant text-[11px]">{d.site_id ?? "—"}</span>
                </button>
              ))}
            </>
          )}
        </div>
      )}
    </div>
  );
}
