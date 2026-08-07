// Package email sends transactional email (invite, password reset) via
// Resend. There is no queue/retry here — a failed send returns an error
// to the caller, who logs it; the underlying action (invite created,
// reset token issued) has already succeeded and is never rolled back for
// an email failure, same principle as recordAction in internal/registry.
package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type Sender interface {
	Send(ctx context.Context, to, subject, html string) error
}

// NewSenderFromEnv returns a real Resend client when RESEND_API_KEY is
// set, otherwise a logging no-op — so invite/reset endpoints work in
// local dev before a Resend domain is verified, without a hard failure
// blocking everything else. Never silently "succeeds" in a way that
// could be mistaken for a delivered email: the no-op logs the full
// content (including the token link) at INFO level specifically so a
// developer can still complete the flow manually.
func NewSenderFromEnv() Sender {
	apiKey := os.Getenv("RESEND_API_KEY")
	from := os.Getenv("EMAIL_FROM_ADDRESS")
	if apiKey == "" || from == "" {
		log.Println("email: RESEND_API_KEY or EMAIL_FROM_ADDRESS not set — using no-op sender (emails logged, not delivered)")
		return &noopSender{}
	}
	return &ResendClient{apiKey: apiKey, from: from, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

type noopSender struct{}

func (n *noopSender) Send(_ context.Context, to, subject, html string) error {
	log.Printf("email (no-op, not delivered): to=%s subject=%q\n%s", to, subject, html)
	return nil
}

// ResendClient talks to https://resend.com's REST API directly — no SDK
// dependency for what is a single POST endpoint.
type ResendClient struct {
	apiKey     string
	from       string
	httpClient *http.Client
}

type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func (r *ResendClient) Send(ctx context.Context, to, subject, html string) error {
	body, err := json.Marshal(resendRequest{From: r.from, To: []string{to}, Subject: subject, HTML: html})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.apiKey)

	res, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		return fmt.Errorf("resend: unexpected status %d", res.StatusCode)
	}
	return nil
}
