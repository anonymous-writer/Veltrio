package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCreatesDefaultAndReadsValues(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.yaml")
	cfg, err := Load(file)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 9999 {
		t.Fatalf("default port=%d", cfg.Port)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatal("default config was not created:", err)
	}
	content := "server:\n  host: \"127.0.0.1\"\n  port: 8088\npaths:\n  data: \"/tmp/veltrio-data\"\n  downloads: \"/tmp/veltrio-downloads\"\n"
	if err := os.WriteFile(file, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err = Load(file)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "127.0.0.1" || cfg.Port != 8088 || cfg.DataDir != "/tmp/veltrio-data" || cfg.DownloadsDir != "/tmp/veltrio-downloads" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
func TestUnknownKeyRejected(t *testing.T) {
	f := filepath.Join(t.TempDir(), "config.yaml")
	_ = os.WriteFile(f, []byte("server:\n  unknown: true\n"), 0600)
	_, err := Load(f)
	if err == nil || !strings.Contains(err.Error(), "unbekannte Einstellung") {
		t.Fatalf("expected unknown setting error, got %v", err)
	}
}
