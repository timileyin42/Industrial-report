// WMO weather codes (the standard Open-Meteo uses) mapped to a short
// label + emoji — full table has ~30 codes, this covers the common
// buckets rather than enumerating every one.
export function describeWeatherCode(code: number): { label: string; icon: string } {
  if (code === 0) return { label: "Clear", icon: "☀️" };
  if (code <= 2) return { label: "Partly Cloudy", icon: "🌤️" };
  if (code === 3) return { label: "Overcast", icon: "☁️" };
  if (code <= 49) return { label: "Foggy", icon: "🌫️" };
  if (code <= 59) return { label: "Drizzle", icon: "🌦️" };
  if (code <= 69) return { label: "Rain", icon: "🌧️" };
  if (code <= 79) return { label: "Snow", icon: "🌨️" };
  if (code <= 99) return { label: "Storm", icon: "⛈️" };
  return { label: "—", icon: "🌡️" };
}

export interface CurrentWeather {
  temperatureC: number;
  windKph: number;
  code: number;
}

// Open-Meteo — free, no API key, no attribution required. Real live
// weather for a site's actual coordinates, not a fabricated widget.
export async function fetchCurrentWeather(lat: number, lng: number): Promise<CurrentWeather> {
  const url = `https://api.open-meteo.com/v1/forecast?latitude=${lat}&longitude=${lng}&current=temperature_2m,weather_code,wind_speed_10m&timezone=auto`;
  const res = await fetch(url);
  if (!res.ok) throw new Error("weather fetch failed");
  const data = await res.json();
  return {
    temperatureC: data.current.temperature_2m,
    windKph: data.current.wind_speed_10m,
    code: data.current.weather_code,
  };
}
