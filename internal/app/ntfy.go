package app

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
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
	if !n.Enabled || strings.TrimSpace(n.URL) == "" {
		return nil
	}
	if n.Debug {
		log.Printf("ntfy: sending notification %q to %s", title, n.URL)
	}
	timeout := n.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.URL, bytes.NewBufferString(message))
	if err != nil {
		return err
	}
	req.Header.Set("Title", title)
	req.Header.Set("Priority", n.Priority)
	req.Header.Set("Tags", "arrow_down")
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	if n.Token != "" && n.Token != "gelöscht" {
		req.Header.Set("Authorization", "Bearer "+n.Token)
	}
	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		if n.Debug {
			log.Printf("ntfy: notification failed: %v", err)
		}
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("ntfy HTTP %s", resp.Status)
		if n.Debug {
			log.Printf("ntfy: notification failed: %v", err)
		}
		return err
	}
	if n.Debug {
		log.Printf("ntfy: notification accepted (HTTP %s)", resp.Status)
	}
	return nil
}
