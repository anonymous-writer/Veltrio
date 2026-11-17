package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseContentRange(t *testing.T) {
	start, total, ok := parseContentRange("bytes 100-199/500")
	if !ok || start != 100 || total != 500 {
		t.Fatalf("unexpected parse: start=%d total=%d ok=%v", start, total, ok)
	}
	if _, _, ok := parseContentRange("items 100-199/500"); ok {
		t.Fatal("accepted invalid unit")
	}
	if _, _, ok := parseContentRange("bytes */500"); ok {
		t.Fatal("accepted unsatisfied range as a partial response")
	}
}

func TestSyncFilesystemMarksMissingAndKeepsRecord(t *testing.T) {
	dir := t.TempDir()
	state := filepath.Join(t.TempDir(), "state.json")
	m, err := NewManagerWithOptions(state, dir, 1, false, false)
	if err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	m.jobs["job-1"] = &job{item: Item{ID: "job-1", Filename: "gone.zip", Status: "completed", Total: 10, Downloaded: 10}}
	_ = m.persistLocked()
	m.mu.Unlock()

	if changed := m.SyncFilesystem(); changed != 1 {
		t.Fatalf("changed=%d, want 1", changed)
	}
	items := m.List()
	if len(items) != 1 || items[0].Status != "missing" {
		t.Fatalf("missing download should remain visible: %+v", items)
	}
	if changed := m.SyncFilesystem(); changed != 0 {
		t.Fatalf("second sync changed=%d, want 0", changed)
	}
}

func TestSyncFilesystemRestoresMissingWhenFileReturns(t *testing.T) {
	dir := t.TempDir()
	state := filepath.Join(t.TempDir(), "state.json")
	m, err := NewManagerWithOptions(state, dir, 1, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "back.zip"), []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	m.jobs["job-1"] = &job{item: Item{ID: "job-1", Filename: "back.zip", Status: "missing", Error: "missing"}}
	_ = m.persistLocked()
	m.mu.Unlock()

	m.SyncFilesystem()
	items := m.List()
	if len(items) != 1 || items[0].Status != "completed" || items[0].Downloaded != 5 || items[0].Total != 5 || items[0].Error != "" {
		t.Fatalf("missing record was not restored: %+v", items)
	}
}

func TestTruncatedDownloadIsNotMarkedCompleted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("four"))
	}))
	defer server.Close()

	dir := t.TempDir()
	m, err := NewManagerWithOptions(filepath.Join(t.TempDir(), "state.json"), dir, 1, false, false)
	if err != nil {
		t.Fatal(err)
	}
	item, err := m.Add(server.URL+"/file.zip", "file.zip")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		items := m.List()
		if len(items) == 1 && (items[0].Status == "error" || items[0].Status == "completed") {
			if items[0].Status == "completed" {
				t.Fatalf("truncated response was incorrectly marked completed: %+v", items[0])
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("download did not finish in time: %+v", m.List())
	_ = item
}
