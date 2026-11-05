package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Ntfy struct {
	Enabled  bool
	Debug    bool
	URL      string
	Token    string
	Priority string
	Timeout  time.Duration
}

func (n Ntfy) Notify(title, message string) error {
	if !n.Enabled {
		return nil
	}
	if err := validateNTFYURL(n.URL); err != nil {
		return fmt.Errorf("ntfy: invalid ntfy_url: %w", err)
	}
	timeout := n.Timeout
	if timeout < time.Second || timeout > 120*time.Second {
		timeout = 10 * time.Second
	}
	priority := validNTFYPriority(n.Priority)
	if n.Debug {
		log.Printf("ntfy: sending download notification title=%q url=%s timeout=%s priority=%s", title, n.URL, timeout, priority)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.URL, bytes.NewBufferString(message))
	if err != nil {
		return fmt.Errorf("ntfy: create request: %w", err)
	}
	req.Header.Set("Title", title)
	req.Header.Set("Priority", priority)
	req.Header.Set("Tags", "arrow_down")
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	if token := strings.TrimSpace(n.Token); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return fmt.Errorf("ntfy: send notification to %s: %w", n.URL, err)
	}
	defer resp.Body.Close()
	if n.Debug {
		log.Printf("ntfy: HTTP response status=%d", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("ntfy: server returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	log.Printf("ntfy: download notification accepted (HTTP %d): %s", resp.StatusCode, title)
	return nil
}

func validNTFYPriority(priority string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "min", "1":
		return "1"
	case "low", "2":
		return "2"
	case "default", "3":
		return "3"
	case "high", "4":
		return "4"
	case "urgent", "5":
		return "5"
	default:
		return "4"
	}
}

func validateNTFYURL(raw string) error {
	u, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return fmt.Errorf("ntfy_url must be an absolute HTTP(S) URL")
	}
	return nil
}
