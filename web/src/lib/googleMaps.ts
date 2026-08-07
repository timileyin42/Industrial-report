// Loads the Google Maps JavaScript API (+ Places library) once via a
// plain injected <script> tag rather than a wrapper package — this is
// the one place in the app that needs the interactive JS API (pin
// placement, address search); MapEmbed.tsx's read-only view elsewhere
// uses the simpler Maps Embed API (iframe), a different product on the
// same key. Both need enabling on whatever key VITE_GOOGLE_MAPS_API_KEY
// points at: Maps JavaScript API + Places API here, Maps Embed API there.
let loadPromise: Promise<void> | null = null;

export function loadGoogleMaps(): Promise<void> {
  const apiKey = import.meta.env.VITE_GOOGLE_MAPS_API_KEY;
  if (!apiKey) {
    return Promise.reject(new Error("VITE_GOOGLE_MAPS_API_KEY is not set"));
  }
  if (window.google?.maps) {
    return Promise.resolve();
  }
  if (loadPromise) {
    return loadPromise;
  }

  loadPromise = new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.src = `https://maps.googleapis.com/maps/api/js?key=${encodeURIComponent(apiKey)}&libraries=places`;
    script.async = true;
    script.onload = () => resolve();
    script.onerror = () => {
      loadPromise = null;
      reject(new Error("Failed to load Google Maps"));
    };
    document.head.appendChild(script);
  });
  return loadPromise;
}
