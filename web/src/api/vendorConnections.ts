import { apiRequest } from "./client";
import {
  VendorProviderSchema,
  VendorConnectionSchema,
  type VendorProvider,
  type VendorConnection,
} from "./types";
import { z } from "zod";

// Public — feeds the Connect-Your-Inverter vendor picker before the
// customer has necessarily finished signing in (see
// internal/httpapi/vendor_connection_handlers.go's listVendorProviders).
export async function listVendorProviders(): Promise<VendorProvider[]> {
  const data = await apiRequest<unknown>("/v1/vendor-providers", { skipAuthRedirect: true });
  return z.object({ providers: z.array(VendorProviderSchema) }).parse(data).providers;
}

export async function createVendorConnection(
  siteId: string,
  input: { provider: string; email: string; password: string }
): Promise<VendorConnection> {
  const data = await apiRequest<unknown>(`/v1/sites/${siteId}/vendor-connections`, {
    method: "POST",
    body: input,
  });
  return VendorConnectionSchema.parse(data);
}

export async function listVendorConnections(siteId: string): Promise<VendorConnection[]> {
  const data = await apiRequest<unknown>(`/v1/sites/${siteId}/vendor-connections`);
  return z.object({ items: z.array(VendorConnectionSchema) }).parse(data).items;
}

export async function revokeVendorConnection(siteId: string, connectionId: number): Promise<void> {
  await apiRequest<void>(`/v1/sites/${siteId}/vendor-connections/${connectionId}`, { method: "DELETE" });
}
