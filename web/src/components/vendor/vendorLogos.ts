import elinterCspLogo from "../../assets/brand/vendors/elinter_csp.png";
import felicitySolarLogo from "../../assets/brand/vendors/felicity_solar.png";
import deyeCloudLogo from "../../assets/brand/vendors/deye_cloud.png";
import sunsynkConnectLogo from "../../assets/brand/vendors/sunsynk_connect.png";

// Each vendor's own real logo, pulled from their official site/app
// store listing — keyed by the same provider name the backend uses, so
// a vendor added later without a logo here just falls back to the
// generic icon (see ConnectVendorFlow) rather than breaking. Shared
// between ConnectVendorFlow (post-signup and the Settings "manage
// connections" page both render it) so the two can never show a
// different logo for the same vendor.
export const VENDOR_LOGOS: Record<string, string> = {
  elinter_csp: elinterCspLogo,
  felicity_solar: felicitySolarLogo,
  deye_cloud: deyeCloudLogo,
  sunsynk_connect: sunsynkConnectLogo,
};
