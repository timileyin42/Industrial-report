package email

import "fmt"

// Plain, inline-styled HTML — no template engine needed for two emails.
// Matches the design system's dark/emerald palette loosely but doesn't
// depend on it: email clients strip most CSS, so this stays simple.

func InviteEmail(acceptURL string) (subject, html string) {
	subject = "You've been invited to Clean Energy Analytics"
	html = fmt.Sprintf(`
		<div style="font-family:sans-serif;max-width:480px;margin:0 auto">
			<h2 style="color:#0b513d">Clean Energy Analytics</h2>
			<p>You've been invited to join the fleet monitoring dashboard.</p>
			<p><a href="%s" style="background:#064e3b;color:#fff;padding:12px 20px;border-radius:4px;text-decoration:none;display:inline-block">Accept invite</a></p>
			<p style="color:#666;font-size:12px">This link expires in 7 days. If you weren't expecting this, you can ignore this email.</p>
		</div>`, acceptURL)
	return subject, html
}

func PasswordResetEmail(resetURL string) (subject, html string) {
	subject = "Reset your Clean Energy Analytics password"
	html = fmt.Sprintf(`
		<div style="font-family:sans-serif;max-width:480px;margin:0 auto">
			<h2 style="color:#0b513d">Clean Energy Analytics</h2>
			<p>We received a request to reset your password.</p>
			<p><a href="%s" style="background:#064e3b;color:#fff;padding:12px 20px;border-radius:4px;text-decoration:none;display:inline-block">Reset password</a></p>
			<p style="color:#666;font-size:12px">This link expires in 1 hour. If you didn't request this, you can ignore this email — your password won't change.</p>
		</div>`, resetURL)
	return subject, html
}
