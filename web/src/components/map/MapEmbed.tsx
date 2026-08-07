import { MapPin } from "lucide-react";
import { EmptyState } from "../feedback/EmptyState";

// Google Maps Embed API via a plain iframe — no @googlemaps/js-api-loader
// dependency needed for a static "show this one location" view. Needs
// VITE_GOOGLE_MAPS_API_KEY (Maps Embed API enabled on that key/project);
// unset falls back to a placeholder rather than a broken iframe, same
// pattern as the backend's no-op email sender when RESEND_API_KEY isn't set.
export function MapEmbed({ lat, lng, label }: { lat: number; lng: number; label?: string }) {
  const apiKey = import.meta.env.VITE_GOOGLE_MAPS_API_KEY;

  if (!apiKey) {
    return (
      <EmptyState
        icon={<MapPin size={40} />}
        title="Map preview unavailable"
        body="Set VITE_GOOGLE_MAPS_API_KEY in web/.env to show this site's location on a map."
      />
    );
  }

  const src = `https://www.google.com/maps/embed/v1/place?key=${encodeURIComponent(apiKey)}&q=${lat},${lng}`;

  return (
    <iframe
      title={label ?? "Site location"}
      src={src}
      className="w-full h-full border-0"
      loading="lazy"
      referrerPolicy="no-referrer-when-downgrade"
    />
  );
}
