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
	if err := n.Notify("Download abgeschlossen", "test.zip"); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer test-token" || gotTitle != "Download abgeschlossen" || gotPriority != "high" || !strings.Contains(gotBody, "test.zip") {
		t.Fatalf("unexpected ntfy request: auth=%q title=%q priority=%q body=%q", gotAuth, gotTitle, gotPriority, gotBody)
	}
}

func TestNtfyDisabledDoesNotSend(t *testing.T) {
	n := Ntfy{Enabled: false, URL: "http://127.0.0.1:1", Timeout: time.Second}
	if err := n.Notify("title", "message"); err != nil {
		t.Fatal(err)
	}
}
