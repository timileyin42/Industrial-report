import { useQuery } from "@tanstack/react-query";
import { fetchCurrentWeather, describeWeatherCode } from "../../lib/weather";

// Real live weather (Open-Meteo, free/keyless) for a real site's actual
// coordinates — not a fabricated readout. Renders nothing useful (a
// graceful placeholder) when no site with coordinates is available yet,
// rather than showing weather for a location that doesn't correspond to
// anything real.
export function WeatherWidget({
  lat,
  lng,
  siteName,
  timezone,
}: {
  lat?: number | null;
  lng?: number | null;
  siteName?: string;
  timezone?: string;
}) {
  const enabled = lat != null && lng != null;
  const { data, isLoading, isError } = useQuery({
    queryKey: ["weather", lat, lng],
    queryFn: () => fetchCurrentWeather(lat!, lng!),
    enabled,
    staleTime: 10 * 60 * 1000,
  });

  // The site's own local date, not the viewer's — avoids showing
  // "yesterday" or "tomorrow" to someone viewing from a timezone where
  // it's already crossed midnight relative to the site.
  const today = new Date().toLocaleDateString(undefined, { month: "short", day: "numeric", weekday: "long", timeZone: timezone });

  if (!enabled) {
    return (
      <div className="glass-card rounded-xl px-5 py-3 text-on-surface-variant font-body-base text-body-base">
        No site location set for weather yet
      </div>
    );
  }

  if (isLoading || isError || !data) {
    return <div className="glass-card rounded-xl px-5 py-3 w-56 h-16 animate-pulse" />;
  }

  const { label, icon } = describeWeatherCode(data.code);

  return (
    <div className="glass-card rounded-xl px-5 py-3 flex items-center gap-4">
      <span className="text-3xl leading-none">{icon}</span>
      <div>
        <p className="font-label-caps text-label-caps text-on-surface-variant uppercase">
          {today}
          {siteName ? ` · ${siteName}` : ""}
        </p>
        <div className="flex items-baseline gap-2">
          <span className="font-data-display-lg text-[22px] text-on-surface">{Math.round(data.temperatureC)}°C</span>
          <span className="font-body-base text-body-base text-on-surface-variant">{label}</span>
        </div>
        <p className="font-body-base text-[12px] text-on-surface-variant">Wind {Math.round(data.windKph)} km/h</p>
      </div>
    </div>
  );
}
