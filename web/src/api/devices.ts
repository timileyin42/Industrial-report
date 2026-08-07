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
