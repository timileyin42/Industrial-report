package registry

import (
	"context"
	"errors"
	"log"
	"net/mail"
	"strings"

	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/email"
)

// DemoRequests handles the marketing site's "Initialize Deployment"
// (Request a Demo) form — internal/httpapi/demo_request_handlers.go's
// public, unauthenticated counterpart. Deliberately not persisted to any
// table: this is a lead-capture relay, not part of the telemetry/audit
// data model the rest of this platform is strict about. A failed send
// here doesn't roll anything back (there's nothing to roll back), same
// principle as recordAction elsewhere in this package.
//
// Three emails go out per submission: (1) the seeded operator account
// (the real login this platform is guaranteed to have) gets notified of
// the request, (2) an optional internal sales/company inbox
// (companyEmail, gated on COMPANY_CONTACT_EMAIL being set) gets a copy,
// and (3) the requester themselves gets an acknowledgement — that one
// is unconditional, since it's their own address from the form, not a
// configured one.
type DemoRequests struct {
	q            *db.Queries
	sender       email.Sender
	companyEmail string
}

func NewDemoRequests(q *db.Queries, sender email.Sender, companyEmail string) *DemoRequests {
	return &DemoRequests{q: q, sender: sender, companyEmail: companyEmail}
}

func (d *DemoRequests) Submit(ctx context.Context, organization, requesterEmail string) error {
	organization = strings.TrimSpace(organization)
	requesterEmail = strings.TrimSpace(requesterEmail)
	if organization == "" {
		return errors.New("organization is required")
	}
	if _, err := mail.ParseAddress(requesterEmail); err != nil {
		return errors.New("a valid email address is required")
	}

	operator, err := d.q.GetEarliestOperator(ctx)
	if err != nil {
		// No operator account exists yet in this environment — still not
		// the requester's fault; log loudly rather than surfacing a 500
		// for what is, from their side, a successful submission.
		log.Printf("demo request: no seeded operator account to notify: %v", err)
	} else {
		subject, html := email.AdminDemoRequestEmail(organization, requesterEmail)
		if err := d.sender.Send(ctx, operator.Email, subject, html); err != nil {
			log.Printf("demo request: failed to notify operator %s: %v", operator.Email, err)
		}
	}

	if d.companyEmail == "" {
		log.Println("demo request: COMPANY_CONTACT_EMAIL not set — skipping company-side notification")
	} else {
		subject, html := email.CompanyDemoRequestEmail(organization, requesterEmail)
		if err := d.sender.Send(ctx, d.companyEmail, subject, html); err != nil {
			log.Printf("demo request: failed to notify company address %s: %v", d.companyEmail, err)
		}
	}

	ackSubject, ackHTML := email.RequesterAcknowledgementEmail(organization)
	if err := d.sender.Send(ctx, requesterEmail, ackSubject, ackHTML); err != nil {
		log.Printf("demo request: failed to send acknowledgement to %s: %v", requesterEmail, err)
	}

	return nil
}
