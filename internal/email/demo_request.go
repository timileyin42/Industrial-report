package email

import (
	"fmt"
	"html"
)

// Same visual language as web/src/index.css's light/glass tokens
// (@theme block) — primary #2f8fe0, on-surface #1e2a3a, surface-dim
// #eef4fb — transcribed here rather than imported, since this package
// has no dependency on the frontend build. The bar-chart mark is drawn
// with inline-styled table cells (web/src/assets/brand/logo-mark.svg's
// three bars/dots), not an <img>, so it renders without depending on
// image hosting or data-URI support in the recipient's email client.
const logoMarkHTML = `
<table role="presentation" cellpadding="0" cellspacing="0" style="display:inline-table;vertical-align:middle">
	<tr style="height:26px">
		<td></td>
		<td></td>
		<td style="width:8px;height:8px;background:#8be0ad;border-radius:50%;"></td>
	</tr>
	<tr style="height:14px">
		<td></td>
		<td style="width:8px;height:8px;background:#8be0ad;border-radius:50%;"></td>
		<td></td>
	</tr>
	<tr style="height:6px">
		<td style="width:8px;height:8px;background:#8be0ad;border-radius:50%;"></td>
		<td></td>
		<td></td>
	</tr>
	<tr>
		<td style="width:8px;height:18px;background:#3f6b52;border-radius:2px;"></td>
		<td style="width:4px"></td>
		<td style="width:8px;height:26px;background:#4f9d6b;border-radius:2px;"></td>
		<td style="width:4px"></td>
		<td style="width:8px;height:38px;background:#57c785;border-radius:2px;"></td>
	</tr>
</table>`

func demoRequestLayout(heading, body string) string {
	return fmt.Sprintf(`
		<div style="font-family:'Plus Jakarta Sans',Arial,sans-serif;max-width:520px;margin:0 auto;background:#eef4fb;padding:32px">
			<div style="background:#ffffff;border:1px solid #e3ecf5;border-radius:16px;padding:32px">
				<div style="margin-bottom:20px">
					%s
					<span style="font-size:16px;font-weight:700;color:#1e2a3a;vertical-align:middle;margin-left:8px">Clean Energy Analytics</span>
				</div>
				<h2 style="color:#1e2a3a;font-size:20px;margin:0 0 16px">%s</h2>
				%s
				<p style="color:#64748b;font-size:11px;margin-top:28px;padding-top:16px;border-top:1px solid #e3ecf5">
					Sent automatically by the Clean Energy Analytics marketing site. This is not a monitored inbox.
				</p>
			</div>
		</div>`, logoMarkHTML, heading, body)
}

// AdminDemoRequestEmail notifies the seeded operator account — see
// registry.DemoRequests.Submit — that a prospect asked for a demo. Sent
// to whichever account cmd/seed-operator created first in this
// environment, not a separate hardcoded address, since that's the one
// real inbox this platform is guaranteed to have.
func AdminDemoRequestEmail(organization, requesterEmail string) (subject, htmlBody string) {
	subject = fmt.Sprintf("New demo request: %s", organization)
	body := fmt.Sprintf(`
		<p style="color:#1e2a3a;font-size:14px;line-height:1.6">
			A prospect submitted the "Initialize Deployment" form on the marketing site.
		</p>
		<table role="presentation" cellpadding="0" cellspacing="0" style="width:100%%;margin:16px 0;background:#f3f8fd;border-radius:12px;padding:16px">
			<tr><td style="font-size:11px;font-weight:700;letter-spacing:0.06em;text-transform:uppercase;color:#64748b;padding:4px 16px">Organization</td></tr>
			<tr><td style="font-size:16px;font-weight:600;color:#1e2a3a;padding:0 16px 12px">%s</td></tr>
			<tr><td style="font-size:11px;font-weight:700;letter-spacing:0.06em;text-transform:uppercase;color:#64748b;padding:4px 16px">Contact Email</td></tr>
			<tr><td style="font-size:16px;font-weight:600;color:#2f8fe0;padding:0 16px 4px">%s</td></tr>
		</table>
		<p style="color:#64748b;font-size:13px">Reply directly to their email to follow up.</p>`,
		html.EscapeString(organization), html.EscapeString(requesterEmail))
	htmlBody = demoRequestLayout("New Demo Request", body)
	return subject, htmlBody
}

// RequesterAcknowledgementEmail is sent back to the prospect who
// submitted the form — confirms their request was received without
// promising a specific response time we can't actually commit to on
// their behalf.
func RequesterAcknowledgementEmail(organization string) (subject, htmlBody string) {
	subject = "We've received your demo request"
	body := fmt.Sprintf(`
		<p style="color:#1e2a3a;font-size:14px;line-height:1.6">
			Thanks for your interest in Clean Energy Analytics on behalf of <strong>%s</strong>.
			Our team has been notified and will reach out shortly to schedule your walkthrough.
		</p>
		<p style="color:#64748b;font-size:13px">
			In the meantime, you can explore how readings are validated using our public
			<a href="https://cleanenergyanalytics.co.uk/sandbox" style="color:#2f8fe0;font-weight:600">Data Sandbox</a> —
			no account required.
		</p>`,
		html.EscapeString(organization))
	htmlBody = demoRequestLayout("Request Received", body)
	return subject, htmlBody
}

// CompanyDemoRequestEmail is the copy sent to the company's own sales
// inbox (COMPANY_CONTACT_EMAIL) — a separate recipient from the seeded
// operator account, per the two-recipient requirement: one real login,
// one real sales/company address, deliberately not conflated into one.
func CompanyDemoRequestEmail(organization, requesterEmail string) (subject, htmlBody string) {
	subject = fmt.Sprintf("New demo request from %s", organization)
	body := fmt.Sprintf(`
		<p style="color:#1e2a3a;font-size:14px;line-height:1.6">
			%s (%s) requested a demo of Clean Energy Analytics through the marketing site's
			"Initialize Deployment" form.
		</p>
		<p style="color:#64748b;font-size:13px">Route this to the sales/onboarding owner for follow-up.</p>`,
		html.EscapeString(organization), html.EscapeString(requesterEmail))
	htmlBody = demoRequestLayout("New Demo Request", body)
	return subject, htmlBody
}
