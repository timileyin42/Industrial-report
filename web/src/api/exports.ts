import { apiRequestRaw } from "./client";

// Filename comes from the browser click, not Content-Disposition parsing —
// the backend endpoints (docs/openapi.yaml) don't guarantee that header,
// so we name the file client-side instead of depending on it.
async function downloadFile(path: string, filename: string): Promise<void> {
  const res = await apiRequestRaw(path);
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

export function downloadSiteTelemetryCSV(siteId: string): Promise<void> {
  return downloadFile(`/v1/sites/${encodeURIComponent(siteId)}/export/telemetry.csv`, `${siteId}-telemetry.csv`);
}

export function downloadSiteSummaryCSV(siteId: string): Promise<void> {
  return downloadFile(`/v1/sites/${encodeURIComponent(siteId)}/export/summary.csv`, `${siteId}-summary.csv`);
}

export function downloadFleetSummaryCSV(): Promise<void> {
  return downloadFile("/v1/fleet/export/summary.csv", "fleet-summary.csv");
}

export function downloadSiteSummaryPDF(siteId: string): Promise<void> {
  return downloadFile(`/v1/sites/${encodeURIComponent(siteId)}/export/summary.pdf`, `${siteId}-summary.pdf`);
}

export function downloadFleetSummaryPDF(): Promise<void> {
  return downloadFile("/v1/fleet/export/summary.pdf", "fleet-summary.pdf");
}
