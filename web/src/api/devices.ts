import { apiRequest } from "./client";
import {
  DeviceSchema,
  DeviceStatusSchema,
  DeviceWithSecretSchema,
  PageSchema,
  type Device,
  type DeviceStatus,
  type DeviceWithSecret,
} from "./types";

export interface RegisterDeviceInput {
  device_id: string;
  site_id: string;
  install_notes?: string;
  inverter_brand?: string;
  inverter_model?: string;
}

export async function listDevices(
  opts: { siteId?: string; cursor?: string; limit?: number } = {}
): Promise<{ items: Device[]; nextCursor?: string }> {
  const data = await apiRequest<unknown>("/v1/devices", {
    query: { site_id: opts.siteId, cursor: opts.cursor, limit: opts.limit ?? 50 },
  });
  const parsed = PageSchema(DeviceSchema).parse(data);
  return { items: parsed.items, nextCursor: parsed.next_cursor };
}

// Pages through every device rather than capping at one page's worth —
// the Dashboard's Fleet Status breakdown needs a true total, not
// whatever the first 200 happen to be (a fleet the size of the load-test
// site alone already exceeds that). Capped at 20 pages (~4000 devices at
// the default 200/page) as a runaway-loop backstop, not an expected limit.
export async function listAllDevices(): Promise<Device[]> {
  const all: Device[] = [];
  let cursor: string | undefined;
  for (let page = 0; page < 20; page++) {
    const { items, nextCursor } = await listDevices({ cursor, limit: 200 });
    all.push(...items);
    if (!nextCursor) break;
    cursor = nextCursor;
  }
  return all;
}

export async function getDevice(deviceId: string): Promise<Device> {
  const data = await apiRequest<unknown>(`/v1/devices/${encodeURIComponent(deviceId)}`);
  return DeviceSchema.parse(data);
}

export async function getDeviceStatus(deviceId: string): Promise<DeviceStatus> {
  const data = await apiRequest<unknown>(`/v1/devices/${encodeURIComponent(deviceId)}/status`);
  return DeviceStatusSchema.parse(data);
}

export async function registerDevice(input: RegisterDeviceInput): Promise<DeviceWithSecret> {
  const data = await apiRequest<unknown>("/v1/devices", { method: "POST", body: input });
  return DeviceWithSecretSchema.parse(data);
}

// The old secret stops working the instant this succeeds — the backend
// invalidates it server-side, not just issues a new one alongside it.
export async function rotateDeviceSecret(deviceId: string): Promise<DeviceWithSecret> {
  const data = await apiRequest<unknown>(`/v1/devices/${encodeURIComponent(deviceId)}/rotate-secret`, { method: "POST" });
  return DeviceWithSecretSchema.parse(data);
}
