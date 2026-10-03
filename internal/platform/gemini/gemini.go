// Package gemini asks Gemini to word assistant replies. It never decides which books to
// recommend: callers pass the picked books in the prompt (BACKEND_PLAN.md section 13).
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client words a prompt. Callers keep their rule-based text when it fails.
type Client interface {
	Word(ctx context.Context, prompt string) (string, error)
}

// New returns the real client, or nil when there is no API key (callers then use
// rule-based replies only).
func New(apiKey, model string, timeout time.Duration) Client {
	if apiKey == "" || model == "" {
		return nil
	}
	return &REST{APIKey: apiKey, Model: model, Timeout: timeout,
		BaseURL: "https://generativelanguage.googleapis.com/v1beta", HTTP: &http.Client{}}
}

// REST calls the Gemini generateContent endpoint.
type REST struct {
	APIKey  string
	Model   string
	Timeout time.Duration
	BaseURL string
	HTTP    *http.Client
}

// ErrEmptyAnswer means Gemini answered with no text.
var ErrEmptyAnswer = errors.New("gemini returned no text")

// Word sends the prompt and returns the first text answer, within the timeout.
func (c *REST) Word(ctx context.Context, prompt string) (string, error) {
	if c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}
	body, _ := json.Marshal(map[string]any{
		"contents": []map[string]any{{"parts": []map[string]string{{"text": prompt}}}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/models/%s:generateContent", c.BaseURL, c.Model), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.APIKey) // in a header, never in the URL, so it cannot reach logs
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini: status %d", resp.StatusCode)
	}
	var out struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("gemini: %w", err)
	}
	if len(out.Candidates) > 0 {
		var sb strings.Builder
		for _, p := range out.Candidates[0].Content.Parts {
			sb.WriteString(p.Text)
		}
		if s := strings.TrimSpace(sb.String()); s != "" {
			return s, nil
		}
	}
	return "", ErrEmptyAnswer
}

// Fake answers with a fixed text or error, and remembers the prompts it saw.
type Fake struct {
	Reply   string
	Err     error
	Prompts []string
}

// Word returns the canned reply.
func (f *Fake) Word(_ context.Context, prompt string) (string, error) {
	f.Prompts = append(f.Prompts, prompt)
	return f.Reply, f.Err
}
