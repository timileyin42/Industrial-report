import { LegalPageLayout, LegalSection } from "../../components/landing/LegalPageLayout";

// Governing law/jurisdiction (Section 10) and the operating entity's
// legal name are left as explicit placeholders — those depend on where
// the business is actually incorporated, which isn't something to guess
// at. Everything else here reflects what the platform actually does
// today: invite-only accounts (no public self-signup), no billing/SLA
// feature yet, role-scoped access, and the real data-integrity posture
// (dedup, validation, audit logging) already built into the backend.
export function TermsOfServicePage() {
  return (
    <LegalPageLayout title="Terms of Service" lastUpdated="7 August 2026">
      <LegalSection heading="1. Acceptance of these terms">
        <p>
          These terms govern access to and use of the Clean Energy Analytics platform ("the Service") by
          any organisation or individual we've granted an account to. By logging in, you agree to these
          terms on behalf of yourself and, where applicable, the organisation you represent.
        </p>
      </LegalSection>

      <LegalSection heading="2. Accounts are invite-only">
        <p>
          There is no public self-registration. An operator-role account creates every other account by
          sending an invitation to an email address; that invitee sets their own password via a link valid
          for a limited time. You're responsible for keeping your credentials confidential and for all
          activity that occurs under your account.
        </p>
        <p>
          Two account roles exist: <strong>operator</strong> (full access to the fleet the organisation
          manages) and <strong>restricted</strong> (scoped to exactly one site). Access is enforced on
          every request the platform receives, not only in what the interface displays.
        </p>
      </LegalSection>

      <LegalSection heading="3. What the Service does">
        <p>
          The Service ingests telemetry from registered solar monitoring devices, stores it, validates it
          against plausibility rules (rejecting or flagging impossible readings — e.g. negative energy, or
          output far above a site's rated capacity), and presents it through dashboards, analytics, and
          exports. Generation and emissions figures are calculated from the data actually received from
          your devices — never fabricated or backfilled with placeholder numbers.
        </p>
      </LegalSection>

      <LegalSection heading="4. Your responsibilities">
        <ul className="list-disc pl-5 space-y-1">
          <li>Provide accurate site information (address, location, system size, country) when registering a site — this directly affects validation thresholds and emissions calculations for that site.</li>
          <li>Keep device credentials secure; a compromised device credential lets someone inject data under that device's identity until it's revoked.</li>
          <li>Only invite users who should legitimately have the level of access you're granting them.</li>
          <li>Not attempt to bypass access controls, rate limits, or authentication.</li>
        </ul>
      </LegalSection>

      <LegalSection heading="5. Data ownership">
        <p>
          You (or the organisation you represent) own the telemetry, site, and device data you submit to
          the Service. We process it to provide the Service and do not use it for any other purpose.
          Deleting a site or device does not retroactively alter historical audit records, which exist to
          support the verification/audit purpose described in our{" "}
          <a href="/privacy" className="text-primary underline">Privacy Policy</a>.
        </p>
      </LegalSection>

      <LegalSection heading="6. Service availability">
        <p>
          We aim for high availability but do not currently offer a formal service-level agreement (SLA)
          or financial remedy for downtime. We'll give reasonable notice of planned maintenance where
          practical.
        </p>
      </LegalSection>

      <LegalSection heading="7. Suspension and termination">
        <p>
          We may suspend or revoke an account or device credential that we reasonably believe is being
          misused, is compromised, or is being used to submit fraudulent data. An organisation's operator
          account holder may request deletion of their organisation's account at any time, subject to any
          data we're required to retain by law or for legitimate audit purposes.
        </p>
      </LegalSection>

      <LegalSection heading="8. Limitation of liability">
        <p>
          The Service is provided "as is." To the maximum extent permitted by applicable law, we're not
          liable for indirect, incidental, or consequential damages arising from use of the Service,
          including decisions made based on generation or emissions figures it reports. Nothing in these
          terms excludes liability that cannot lawfully be excluded under Nigerian or UK law.
        </p>
      </LegalSection>

      <LegalSection heading="9. Changes to these terms">
        <p>
          We'll update the date at the top of this page when these terms change. Continued use of the
          Service after a change takes effect constitutes acceptance of the updated terms.
        </p>
      </LegalSection>

      <LegalSection heading="10. Governing law">
        <p>
          <em>
            [Placeholder — to be finalised based on the operating entity's jurisdiction of incorporation:
            these terms will be governed by the laws of either Nigeria or England and Wales, with the
            courts of that jurisdiction having exclusive jurisdiction over any dispute.]
          </em>
        </p>
      </LegalSection>

      <LegalSection heading="11. Contact">
        <p>
          Questions about these terms can be sent to{" "}
          <a href="mailto:legal@cleanenergyanalytics.co.uk" className="text-primary underline">
            legal@cleanenergyanalytics.co.uk
          </a>
          .
        </p>
      </LegalSection>
    </LegalPageLayout>
  );
}
