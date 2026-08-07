import { z } from "zod";

// Mirrors internal/domain.Role — literal values, not paraphrased, since
// the backend and any future display logic must agree on the exact string.
export const RoleSchema = z.enum(["operator", "restricted"]);
export type Role = z.infer<typeof RoleSchema>;

export const LoginResponseSchema = z.object({
  token: z.string(),
  expires_at: z.string(),
  role: RoleSchema,
  site_id: z.string().nullable().optional(),
});
export type LoginResponse = z.infer<typeof LoginResponseSchema>;

export const SiteSchema = z.object({
  site_id: z.string(),
  name: z.string().nullable().optional(),
  cohort_id: z.string().nullable().optional(),
  address: z.string().nullable().optional(),
  gps_lat: z.number().nullable().optional(),
  gps_lng: z.number().nullable().optional(),
  inverter_make_model: z.string().nullable().optional(),
  system_size_kw: z.number().nullable().optional(),
  timezone: z.string(),
  created_at: z.string(),
});
export type Site = z.infer<typeof SiteSchema>;

export const PageSchema = <T extends z.ZodTypeAny>(item: T) =>
  z.object({
    items: z.array(item),
    next_cursor: z.string().optional(),
  });

export const DeviceSchema = z.object({
  device_id: z.string(),
  site_id: z.string().nullable().optional(),
  revoked_at: z.string().nullable().optional(),
  last_seen_at: z.string().nullable().optional(),
  created_at: z.string(),
  secret_last_rotated_at: z.string(),
  install_notes: z.string().nullable().optional(),
});
export type Device = z.infer<typeof DeviceSchema>;

export const DeviceWithSecretSchema = DeviceSchema.extend({
  secret: z.string(),
});
export type DeviceWithSecret = z.infer<typeof DeviceWithSecretSchema>;

export const DeviceStatusSchema = z.object({
  device_id: z.string(),
  last_seen_at: z.string().nullable().optional(),
  last_contact_at: z.string().nullable().optional(),
  online: z.boolean(),
  data_gap: z.boolean(),
  revoked: z.boolean(),
});
export type DeviceStatus = z.infer<typeof DeviceStatusSchema>;

export const FleetSummarySchema = z.object({
  total_sites: z.number(),
  total_devices: z.number(),
  online_devices: z.number(),
  total_capacity_kw: z.number().nullable().optional(),
});
export type FleetSummary = z.infer<typeof FleetSummarySchema>;

export const TelemetryPointSchema = z.object({
  ts: z.string(),
  device_id: z.string(),
  power_kw: z.number(),
  energy_kwh_total: z.number(),
  voltage_v: z.number().nullable().optional(),
  status: z.enum(["ok", "fault", "offline"]),
});
export type TelemetryPoint = z.infer<typeof TelemetryPointSchema>;

export const EnergyPointSchema = z.object({
  period_start: z.string(),
  energy_kwh: z.number(),
  reading_count: z.number(),
  backfilled_count: z.number(),
});
export const EnergySeriesSchema = z.object({
  unit: z.string(),
  period: z.string(),
  points: z.array(EnergyPointSchema),
  cumulative_kwh: z.number(),
});
export type EnergySeries = z.infer<typeof EnergySeriesSchema>;

export const YieldPointSchema = z.object({
  period_start: z.string(),
  energy_kwh: z.number(),
  system_size_kw: z.number(),
  specific_yield_kwh_per_kwp: z.number(),
});
export const YieldSeriesSchema = z.object({
  unit: z.string(),
  period: z.string(),
  points: z.array(YieldPointSchema),
});
export type YieldSeries = z.infer<typeof YieldSeriesSchema>;

export const PeakPointSchema = z.object({
  day: z.string(),
  peak_power_kw: z.number(),
  occurred_at: z.string().nullable().optional(),
});
export const PeakSeriesSchema = z.object({
  unit: z.string(),
  points: z.array(PeakPointSchema),
});
export type PeakSeries = z.infer<typeof PeakSeriesSchema>;

export const CapacityFactorPointSchema = z.object({
  period_start: z.string(),
  energy_kwh: z.number(),
  theoretical_max_kwh: z.number(),
  capacity_factor_pct: z.number(),
});
export const CapacityFactorSeriesSchema = z.object({
  definition: z.string(),
  period: z.string(),
  points: z.array(CapacityFactorPointSchema),
});
export type CapacityFactorSeries = z.infer<typeof CapacityFactorSeriesSchema>;

// Mirrors internal/registry.EmissionFactor — kg_co2_per_kwh must never be
// hand-entered on the frontend beyond this exact setup form (see
// registry/emissions.go: "never invented" applies here too).
export const EmissionFactorSchema = z.object({
  id: z.number(),
  kg_co2_per_kwh: z.number(),
  country: z.string(),
  source: z.string(),
  effective_from: z.string(),
});
export type EmissionFactor = z.infer<typeof EmissionFactorSchema>;

export const EmissionPointSchema = z.object({
  period_start: z.string(),
  energy_kwh: z.number(),
  kg_co2: z.number(),
});
export const EmissionsSeriesSchema = z.object({
  unit_kg: z.string(),
  unit_tonnes: z.string(),
  period: z.string(),
  emission_factor: EmissionFactorSchema,
  points: z.array(EmissionPointSchema),
  cumulative_lifetime_co2_tonnes: z.number(),
});
export type EmissionsSeries = z.infer<typeof EmissionsSeriesSchema>;

export const HistoryComparisonSchema = z.object({
  current_period_start: z.string(),
  previous_period_start: z.string(),
  current_energy_kwh: z.number(),
  previous_energy_kwh: z.number(),
  change_pct: z.number().nullable().optional(),
});
export type HistoryComparison = z.infer<typeof HistoryComparisonSchema>;

export const FleetComparisonSchema = z.object({
  site_id: z.string(),
  site_energy_kwh: z.number(),
  fleet_avg_kwh: z.number(),
  percentile_rank: z.number(),
  site_count: z.number(),
});
export type FleetComparison = z.infer<typeof FleetComparisonSchema>;

export const SegmentStatSchema = z.object({
  segment_key: z.string(),
  site_count: z.number(),
  total_energy_kwh: z.number(),
  avg_energy_kwh: z.number(),
});
export const SegmentResultSchema = z.object({
  segment_by: z.string(),
  items: z.array(SegmentStatSchema),
  next_cursor: z.string().optional(),
  note: z.string().optional(),
});
export type SegmentResult = z.infer<typeof SegmentResultSchema>;

export const TrendPointSchema = z.object({
  period_start: z.string(),
  total_capacity_kw: z.number(),
  site_count: z.number(),
  total_energy_kwh: z.number(),
  mom_change_pct: z.number().nullable().optional(),
});
export const FleetTrendsSchema = z.object({
  period: z.string(),
  points: z.array(TrendPointSchema),
});
export type FleetTrends = z.infer<typeof FleetTrendsSchema>;

export const AnomalyFlagSchema = z.object({
  site_id: z.string(),
  day: z.string(),
  energy_kwh: z.number(),
  baseline_kwh: z.number(),
  drop_fraction: z.number(),
});
export const AnomalyResultSchema = z.object({
  definition: z.string(),
  flags: z.array(AnomalyFlagSchema),
});
export type AnomalyResult = z.infer<typeof AnomalyResultSchema>;

export const SiteHealthSchema = z.object({
  site_id: z.string(),
  site_name: z.string().nullable().optional(),
  total_devices: z.number(),
  online_devices: z.number(),
  coverage_pct: z.number(),
  worst_last_seen_at: z.string().nullable().optional(),
});
export const FleetHealthSchema = z.object({
  generated_at: z.string(),
  online_threshold_minutes: z.number(),
  expected_interval_minutes: z.number(),
  coverage_window_hours: z.number(),
  fleet: z.object({
    total_sites: z.number(),
    total_devices: z.number(),
    online_devices: z.number(),
    devices_reporting_pct: z.number(),
    coverage_pct: z.number(),
  }),
  sites: PageSchema(SiteHealthSchema),
});
export type FleetHealth = z.infer<typeof FleetHealthSchema>;
export type SiteHealth = z.infer<typeof SiteHealthSchema>;

export const AuditEntrySchema = z.object({
  id: z.number(),
  actor_email: z.string().nullable().optional(),
  action: z.string(),
  target_type: z.string().nullable().optional(),
  target_id: z.string().nullable().optional(),
  metadata: z.record(z.string(), z.unknown()).nullable().optional(),
  created_at: z.string(),
});
export type AuditEntry = z.infer<typeof AuditEntrySchema>;

export const IngestionAuditEntrySchema = z.object({
  id: z.number(),
  device_id: z.string(),
  site_id: z.string().nullable().optional(),
  raw_payload: z.record(z.string(), z.unknown()).nullable().optional(),
  received_at: z.string(),
  processed: z.boolean(),
  error: z.string().nullable().optional(),
});
export type IngestionAuditEntry = z.infer<typeof IngestionAuditEntrySchema>;

export const ExportJobSchema = z.object({
  id: z.number(),
  job_type: z.enum(["site_telemetry_csv", "site_summary_csv", "fleet_summary_csv"]),
  site_id: z.string().nullable().optional(),
  status: z.enum(["pending", "running", "completed", "failed"]),
  error: z.string().nullable().optional(),
  created_at: z.string(),
  completed_at: z.string().nullable().optional(),
  download_url: z.string().nullable().optional(),
});
export type ExportJob = z.infer<typeof ExportJobSchema>;
export type ExportJobType = ExportJob["job_type"];

// ApiError carries the HTTP status so callers can distinguish 401 (global
// redirect) from 403 (AccessDenied) from everything else (ErrorState) —
// see src/api/client.ts.
export class ApiError extends Error {
  status: number;
  body: unknown;

  constructor(status: number, message: string, body?: unknown) {
    super(message);
    this.status = status;
    this.body = body;
  }
}
