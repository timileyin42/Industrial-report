import { useEffect, useRef, useState } from "react";
import { loadGoogleMaps } from "../../lib/googleMaps";

interface LocationPickerProps {
  lat: number | null;
  lng: number | null;
  onChange: (lat: number, lng: number, address?: string) => void;
}

const inputClass =
  "w-full bg-surface-dim border border-outline-variant text-on-surface font-body-base rounded-sm py-2.5 px-4 focus:border-primary focus:ring-1 focus:ring-primary outline-none placeholder:text-outline-variant/50";
const monoInputClass =
  "w-full bg-surface-dim border border-outline-variant text-on-surface font-data-mono-sm text-data-mono-sm rounded-sm py-2.5 px-4 focus:border-primary focus:ring-1 focus:ring-primary outline-none placeholder:text-outline-variant/50";

// Search a place or click the map to drop a pin — replaces hand-typed
// lat/lng. Falls back to plain numeric inputs (the old behavior) if
// VITE_GOOGLE_MAPS_API_KEY isn't set or the script fails to load, same
// graceful-degradation pattern as MapEmbed.tsx, rather than blocking site
// creation on a missing key.
export function LocationPicker({ lat, lng, onChange }: LocationPickerProps) {
  const mapDivRef = useRef<HTMLDivElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const mapRef = useRef<google.maps.Map | null>(null);
  const markerRef = useRef<google.maps.Marker | null>(null);
  const geocoderRef = useRef<google.maps.Geocoder | null>(null);
  const [available, setAvailable] = useState<boolean | null>(null);

  useEffect(() => {
    let cancelled = false;
    loadGoogleMaps()
      .then(() => {
        if (cancelled || !mapDivRef.current) return;
        setAvailable(true);

        const center = lat != null && lng != null ? { lat, lng } : { lat: 20, lng: 0 };
        const zoom = lat != null && lng != null ? 14 : 2;
        const map = new google.maps.Map(mapDivRef.current, {
          center,
          zoom,
          mapTypeControl: false,
          streetViewControl: false,
          fullscreenControl: false,
        });
        mapRef.current = map;
        geocoderRef.current = new google.maps.Geocoder();

        if (lat != null && lng != null) {
          markerRef.current = new google.maps.Marker({ position: center, map, draggable: true });
          markerRef.current.addListener("dragend", () => {
            const pos = markerRef.current!.getPosition();
            if (pos) onChange(pos.lat(), pos.lng());
          });
        }

        map.addListener("click", (e: google.maps.MapMouseEvent) => {
          if (!e.latLng) return;
          placeMarker(e.latLng.lat(), e.latLng.lng());
        });

        if (searchInputRef.current) {
          const autocomplete = new google.maps.places.Autocomplete(searchInputRef.current, {
            fields: ["geometry", "formatted_address"],
          });
          autocomplete.addListener("place_changed", () => {
            const place = autocomplete.getPlace();
            const loc = place.geometry?.location;
            if (!loc) return;
            map.panTo(loc);
            map.setZoom(14);
            placeMarker(loc.lat(), loc.lng(), place.formatted_address);
          });
        }
      })
      .catch(() => {
        if (!cancelled) setAvailable(false);
      });
    return () => {
      cancelled = true;
    };
    // Intentionally runs once — re-centering on external lat/lng changes
    // after the map exists isn't needed for a create-site form.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function placeMarker(newLat: number, newLng: number, knownAddress?: string) {
    const map = mapRef.current;
    if (!map) return;
    const position = { lat: newLat, lng: newLng };
    if (markerRef.current) {
      markerRef.current.setPosition(position);
    } else {
      markerRef.current = new google.maps.Marker({ position, map, draggable: true });
      markerRef.current.addListener("dragend", () => {
        const pos = markerRef.current!.getPosition();
        if (pos) onChange(pos.lat(), pos.lng());
      });
    }

    if (knownAddress) {
      onChange(newLat, newLng, knownAddress);
      return;
    }
    geocoderRef.current?.geocode({ location: position }, (results, status) => {
      if (status === "OK" && results?.[0]) {
        onChange(newLat, newLng, results[0].formatted_address);
      } else {
        onChange(newLat, newLng);
      }
    });
  }

  if (available === false) {
    return (
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-label-caps font-label-caps text-on-surface-variant mb-2">LATITUDE</label>
          <input
            className={monoInputClass}
            placeholder="0.000000"
            type="number"
            step="any"
            value={lat ?? ""}
            onChange={(e) => onChange(Number(e.target.value), lng ?? 0)}
          />
        </div>
        <div>
          <label className="block text-label-caps font-label-caps text-on-surface-variant mb-2">LONGITUDE</label>
          <input
            className={monoInputClass}
            placeholder="0.000000"
            type="number"
            step="any"
            value={lng ?? ""}
            onChange={(e) => onChange(lat ?? 0, Number(e.target.value))}
          />
        </div>
        <p className="col-span-2 text-[10px] text-on-surface-variant">
          Map picker unavailable — set VITE_GOOGLE_MAPS_API_KEY to search or click a location instead of typing coordinates.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <input ref={searchInputRef} type="text" placeholder="Search for an address…" className={inputClass} disabled={available !== true} />
      <div ref={mapDivRef} className="w-full h-[220px] rounded-sm bg-surface-dim border border-outline-variant" />
      {lat != null && lng != null ? (
        <p className="text-[10px] font-data-mono-sm text-data-mono-sm text-on-surface-variant">
          {lat.toFixed(6)}, {lng.toFixed(6)}
        </p>
      ) : (
        <p className="text-[10px] text-on-surface-variant">Search above or click the map to drop a pin.</p>
      )}
    </div>
  );
}
