import { LegalPageLayout, LegalSection } from "../../components/landing/LegalPageLayout";

// Grounded in what this platform actually stores (see internal/db/schema
// — users: email/password_hash/role/site_id only; sites: name/address/
// gps/inverter/system_size/timezone/country; user_action_audit_log:
// actor_user_id/action/target, no IP or device fingerprinting) and the
// two regimes both operating jurisdictions fall under: Nigeria's Data
// Protection Act 2023 (NDPA, enforced by the Nigeria Data Protection
// Commission — NDPC) and the UK GDPR + Data Protection Act 2018
// (enforced by the ICO). This is drafted to be accurate about the real
// system, not a generic boilerplate template — but per CLAUDE.md's own
// rule on this exact topic ("confirm specific retention/handling
// requirements with the client rather than assuming"), have this
// reviewed by qualified counsel in both jurisdictions before treating it
// as final/binding, particularly the retention window (currently a
// provisional 2-year default on raw telemetry, see docs/retention.md)
// and the contact details below, which are placeholders.
export function PrivacyPolicyPage() {
  return (
    <LegalPageLayout title="Privacy Policy" lastUpdated="7 August 2026">
      <LegalSection heading="1. Who we are">
        <p>
          Clean Energy Analytics ("we", "us", "our") operates a solar fleet monitoring platform used by
          installers and asset operators in Nigeria and the United Kingdom. This policy explains what
          personal data we collect through that platform, why, and what rights you have over it.
        </p>
      </LegalSection>

      <LegalSection heading="2. What personal data we collect">
        <p>We collect substantially less personal data than most SaaS products, by design:</p>
        <ul className="list-disc pl-5 space-y-1">
          <li>
            <strong>Account data.</strong> When an operator invites you as a user, we store your email
            address, a hashed (never plaintext) password, your assigned role, and — for restricted accounts
            — the single site you're scoped to.
          </li>
          <li>
            <strong>Site address information.</strong> A site's street address and GPS coordinates are
            entered by the operator who registers that site. Where a site corresponds to a private
            property, this address may identify or help identify the property's owner or occupant.
          </li>
          <li>
            <strong>Platform activity.</strong> We log which user performed which administrative action
            (e.g. "created a site," "revoked a device") and when — never your IP address, browsing
            behaviour, or location beyond what's described above.
          </li>
        </ul>
        <p>
          Device telemetry itself — power output, cumulative energy, voltage, device status — is generation
          data about hardware, not personal data about a person.
        </p>
      </LegalSection>

      <LegalSection heading="3. Why we process it, and our legal basis">
        <ul className="list-disc pl-5 space-y-1">
          <li>To provide the account you were invited to use (performance of a contract / legitimate interest of the operator who invited you).</li>
          <li>To maintain an accurate administrative audit trail of who changed what on the platform (legitimate interest, and — for verification-grade generation records — a legal obligation once emissions/audit reporting requires it).</li>
          <li>To keep the platform secure (legitimate interest: detecting misuse, revoked-device enforcement, abuse prevention).</li>
        </ul>
        <p>We do not sell personal data, and we do not use it for advertising or profiling.</p>
      </LegalSection>

      <LegalSection heading="4. Where your data is stored and who can see it">
        <p>
          Data is stored on infrastructure we operate and is not shared with third parties except where
          strictly necessary to run the service (e.g. a transactional email provider to deliver invite/
          password-reset links, and object storage for files you explicitly export). Access within the
          platform is role-based: a restricted account can only ever see data for the one site it's scoped
          to, enforced on every request — not just hidden in the interface.
        </p>
        <p>
          If personal data is transferred across a border as part of running this service (for example,
          between Nigeria and the UK, or to hosting infrastructure located outside either country), we put
          appropriate safeguards in place consistent with the NDPA and UK GDPR's cross-border transfer
          requirements.
        </p>
      </LegalSection>

      <LegalSection heading="5. How long we keep it">
        <p>
          Account records are kept for as long as your account is active, plus a reasonable period
          afterward for audit purposes. Raw device telemetry is currently retained for up to two years,
          after which it is automatically deleted — this window may be extended where a longer
          verification/audit period is required for emissions reporting. Aggregated/summarised generation
          data (daily rollups) is kept longer, since it no longer carries the same granularity.
        </p>
      </LegalSection>

      <LegalSection heading="6. Your rights">
        <p>Under the NDPA (Nigeria) and UK GDPR (United Kingdom), you have the right to:</p>
        <ul className="list-disc pl-5 space-y-1">
          <li>Ask what personal data we hold about you, and get a copy of it (access).</li>
          <li>Ask us to correct inaccurate data (rectification).</li>
          <li>Ask us to delete your data, subject to any legal retention obligations (erasure).</li>
          <li>Object to or restrict certain processing.</li>
          <li>Receive your data in a portable format.</li>
          <li>Lodge a complaint with your national regulator — the Nigeria Data Protection Commission (NDPC) or the UK Information Commissioner's Office (ICO) — if you believe we've mishandled your data.</li>
        </ul>
        <p>To exercise any of these, contact us using the details in Section 8.</p>
      </LegalSection>

      <LegalSection heading="7. Security">
        <p>
          See our <a href="/security" className="text-primary underline">Security page</a> for the specific
          technical measures we take to protect this data — encrypted transport, per-device credentials,
          hashed secrets, and role-based access control among them.
        </p>
      </LegalSection>

      <LegalSection heading="8. Contact">
        <p>
          Questions about this policy, or requests to exercise your rights, can be sent to{" "}
          <a href="mailto:privacy@cleanenergyanalytics.co.uk" className="text-primary underline">
            privacy@cleanenergyanalytics.co.uk
          </a>
          .
        </p>
      </LegalSection>

      <LegalSection heading="9. Changes to this policy">
        <p>
          We'll update the date at the top of this page whenever this policy changes, and where a change
          is significant, we'll take reasonable steps to let active users know.
        </p>
      </LegalSection>
    </LegalPageLayout>
  );
}
