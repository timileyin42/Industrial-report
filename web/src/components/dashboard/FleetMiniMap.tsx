import { useEffect, useRef, useState } from "react";
import { loadGoogleMaps } from "../../lib/googleMaps";
import { EmptyState } from "../feedback/EmptyState";
import type { Site, SiteHealth } from "../../api/types";

// Shared by MapViewPage (full-size) and the Dashboard's map preview
// panel (compact) — same marker logic, so "click a marker on the
// Dashboard preview" and "click one on the full Map View" behave
// identically rather than maintaining two Google Maps integrations.
export function FleetMiniMap({
  sites,
  healthBySite,
  height = 220,
  zoom,
  onSiteClick,
  compact,
}: {
  sites: Site[];
  healthBySite: Map<string, SiteHealth>;
  height?: number | string;
  zoom?: number;
  onSiteClick?: (siteId: string) => void;
  compact?: boolean;
}) {
  const mapDivRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<google.maps.Map | null>(null);
  const markersRef = useRef<google.maps.Marker[]>([]);
  const [mapsAvailable, setMapsAvailable] = useState<boolean | null>(null);

  useEffect(() => {
    let cancelled = false;
    loadGoogleMaps()
      .then(() => {
        if (!cancelled) setMapsAvailable(true);
      })
      .catch(() => {
        if (!cancelled) setMapsAvailable(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const located = sites.filter((s) => s.gps_lat != null && s.gps_lng != null);

  useEffect(() => {
    if (!mapsAvailable || !mapDivRef.current || located.length === 0) return;

    if (!mapRef.current) {
      mapRef.current = new google.maps.Map(mapDivRef.current, {
        center: { lat: located[0].gps_lat!, lng: located[0].gps_lng! },
        zoom: zoom ?? 6,
        mapTypeControl: false,
        streetViewControl: false,
        fullscreenControl: false,
        zoomControl: !compact,
        gestureHandling: compact ? "none" : "auto",
      });
    }
    const map = mapRef.current;

    markersRef.current.forEach((m) => m.setMap(null));
    markersRef.current = [];

    const bounds = new google.maps.LatLngBounds();
    for (const site of located) {
      const health = healthBySite.get(site.site_id);
      // Same three-color status semantics used everywhere else in this
      // app (green/amber/red = online/degraded/offline) — never a new ad
      // hoc color for "no health data yet" (falls back to amber).
      let color = "#f2a93b";
      if (health) {
        if (health.online_devices === health.total_devices && health.total_devices > 0) color = "#1a9c6b";
        else if (health.online_devices === 0) color = "#e4483a";
      }
      const position = { lat: site.gps_lat!, lng: site.gps_lng! };
      const marker = new google.maps.Marker({
        position,
        map,
        title: site.name ?? site.site_id,
        icon: {
          path: google.maps.SymbolPath.CIRCLE,
          scale: compact ? 7 : 9,
          fillColor: color,
          fillOpacity: 1,
          strokeColor: "#ffffff",
          strokeWeight: 2,
        },
      });
      if (onSiteClick) marker.addListener("click", () => onSiteClick(site.site_id));
      markersRef.current.push(marker);
      bounds.extend(position);
    }
    if (located.length > 1) map.fitBounds(bounds);
  }, [mapsAvailable, located, healthBySite, onSiteClick, zoom, compact]);

  if (located.length === 0) {
    return (
      <div style={{ height }}>
        <EmptyState compact={compact} title="No sites mapped yet" body="Add a GPS location to a site to see it here." />
      </div>
    );
  }
  if (mapsAvailable === false) {
    return (
      <div style={{ height }}>
        <EmptyState compact={compact} title="Map unavailable" body="VITE_GOOGLE_MAPS_API_KEY isn't set." />
      </div>
    );
  }
  return <div ref={mapDivRef} style={{ height }} className="w-full rounded-2xl overflow-hidden" />;
}
