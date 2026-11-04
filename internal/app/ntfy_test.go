package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNtfyNotifySendsCompletionMessage(t *testing.T) {
	var gotAuth, gotTitle, gotPriority, gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotTitle = r.Header.Get("Title")
		gotPriority = r.Header.Get("Priority")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()
	n := Ntfy{Enabled: true, URL: ts.URL, Token: "test-token", Priority: "high", Timeout: 2 * time.Second}
	if err := n.Notify("Veltrio: Download abgeschlossen", "test.zip"); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer test-token" || gotTitle != "Veltrio: Download abgeschlossen" || gotPriority != "4" || !strings.Contains(gotBody, "test.zip") {
		t.Fatalf("unexpected ntfy request: auth=%q title=%q priority=%q body=%q", gotAuth, gotTitle, gotPriority, gotBody)
	}
}

func TestNtfyDisabledDoesNotSend(t *testing.T) {
	n := Ntfy{Enabled: false, URL: "http://127.0.0.1:1", Timeout: time.Second}
	if err := n.Notify("title", "message"); err != nil {
		t.Fatal(err)
	}
}

func TestValidateNTFYURL(t *testing.T) {
	for _, tc := range []struct {
		name, url string
		wantErr   bool
	}{
		{name: "https topic", url: "https://ntfy.sh/veltrio"},
		{name: "self hosted", url: "http://127.0.0.1:2586/PI-Micha-Info"},
		{name: "missing scheme", url: "127.0.0.1:2586/topic", wantErr: true},
		{name: "unsupported scheme", url: "ftp://example.com/topic", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateNTFYURL(tc.url); (err != nil) != tc.wantErr {
				t.Fatalf("validateNTFYURL(%q) error=%v wantErr=%v", tc.url, err, tc.wantErr)
			}
		})
	}
}

func TestValidNTFYPriority(t *testing.T) {
	for input, want := range map[string]string{"min": "1", "low": "2", "default": "3", "high": "4", "urgent": "5", "unknown": "4"} {
		if got := validNTFYPriority(input); got != want {
			t.Errorf("validNTFYPriority(%q)=%q want %q", input, got, want)
		}
	}
}
