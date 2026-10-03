package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func postJSON(ctx context.Context, c *http.Client, url string, headers map[string]string, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("send email: provider answered %d", resp.StatusCode)
	}
	return nil
}

// Resend sends through the Resend API.
type Resend struct {
	APIKey string
	From   string
	URL    string
	Client *http.Client
}

// SendOTP sends the code.
func (r *Resend) SendOTP(ctx context.Context, to, code, purpose string) error {
	subject, text := subjectAndText(code, purpose)
	return postJSON(ctx, r.Client, r.URL, map[string]string{"Authorization": "Bearer " + r.APIKey},
		map[string]any{"from": r.From, "to": []string{to}, "subject": subject, "text": text})
}

// Brevo sends through the Brevo transactional API.
type Brevo struct {
	APIKey string
	From   string
	URL    string
	Client *http.Client
}

// SendOTP sends the code.
func (b *Brevo) SendOTP(ctx context.Context, to, code, purpose string) error {
	subject, text := subjectAndText(code, purpose)
	name, addr := parseFrom(b.From)
	return postJSON(ctx, b.Client, b.URL, map[string]string{"api-key": b.APIKey},
		map[string]any{
			"sender":      map[string]string{"name": name, "email": addr},
			"to":          []map[string]string{{"email": to}},
			"subject":     subject,
			"textContent": text,
		})
}
