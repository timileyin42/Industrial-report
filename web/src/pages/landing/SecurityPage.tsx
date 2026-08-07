import { LegalPageLayout, LegalSection } from "../../components/landing/LegalPageLayout";

// Every claim here maps to something actually implemented, not aspirational
// copy — cross-referenced against CLAUDE.md's security section, AGENTS.md,
// docs/tls.md, and the actual backend code (per-device MQTT creds +
// mosquitto/config/acl.conf, secret_hash never plaintext, CORS allow-list
// with no wildcard, role-based access enforced server-side, ingestion +
// user-action audit logs). Where something is a known gap rather than
// finished, it's stated as such (e.g. mutual-TLS on the broker) rather
// than implied to already be in place.
export function SecurityPage() {
  return (
    <LegalPageLayout title="Security" lastUpdated="7 August 2026">
      <LegalSection heading="Encryption in transit">
        <p>
          Traffic between your browser and the platform, and between the platform and Cloudflare's edge, is
          encrypted with TLS. Every device that publishes telemetry connects to our MQTT broker over its
          own authenticated connection; a TLS listener is available for devices that support it, alongside
          the plaintext listener used during initial device provisioning on a trusted local network.
        </p>
      </LegalSection>

      <LegalSection heading="Device identity and revocation">
        <p>
          Every device authenticates with its own credential — no device is trusted by network location
          alone. Per-device access rules at the broker mean a leaked credential for one device cannot be
          used to publish data for another device or site. Revoking a device is enforced at two independent
          layers — the broker's own access rules, and an application-level check on every incoming
          message — so a revoked device cannot inject data through either path alone. Revoked devices'
          messages are still recorded for forensic review; they're just never written into live telemetry.
        </p>
      </LegalSection>

      <LegalSection heading="Access control">
        <p>
          Every account has exactly one role: full access across an organisation's fleet, or restricted
          access to a single site. This is enforced on every API request the platform serves, not only in
          what a restricted account's interface happens to display — a restricted account that tries to
          request another site's data by editing a URL directly still gets denied.
        </p>
      </LegalSection>

      <LegalSection heading="Secrets">
        <p>
          Device credentials are stored as one-way hashes, never as plaintext, the same way account
          passwords are. A newly issued or rotated device secret is shown once, at the moment of
          issuance — it cannot be retrieved again afterward, only rotated to a new one.
        </p>
      </LegalSection>

      <LegalSection heading="Audit logging">
        <p>
          Two independent audit trails are kept: one recording every telemetry message received — including
          ones that failed validation — for data-quality forensics, and a separate one recording
          administrative actions (who created a site, revoked a device, changed an emission factor, and
          when). Neither log records IP addresses or browsing behaviour; both exist to answer "what
          happened and who did it," not to track users.
        </p>
      </LegalSection>

      <LegalSection heading="Data validation">
        <p>
          Incoming readings are checked against per-site plausibility bounds before being accepted —
          implausible values (negative energy, output far above a site's rated capacity, non-zero output
          overnight) are rejected or flagged rather than silently stored. Duplicate deliveries of the same
          reading are deduplicated; out-of-order or backfilled readings are handled explicitly rather than
          assumed to arrive in order.
        </p>
      </LegalSection>

      <LegalSection heading="Network-level protections">
        <p>
          Cross-origin requests are restricted to an explicit list of known dashboard origins — never a
          wildcard. Public-facing endpoints are rate-limited to reduce brute-force and abuse risk.
        </p>
      </LegalSection>

      <LegalSection heading="What we're still hardening">
        <p>
          In the interest of not overstating our posture: mutual TLS (requiring devices to present their
          own certificate, on top of username/password authentication) isn't enabled on the broker yet.
          We treat this list as something to keep extending, not a finished state.
        </p>
      </LegalSection>

      <LegalSection heading="Reporting a security issue">
        <p>
          If you believe you've found a security vulnerability, please report it to{" "}
          <a href="mailto:security@cleanenergyanalytics.co.uk" className="text-primary underline">
            security@cleanenergyanalytics.co.uk
          </a>{" "}
          rather than filing it publicly. We'll acknowledge reports and work to address genuine issues
          promptly.
        </p>
      </LegalSection>
    </LegalPageLayout>
  );
}
