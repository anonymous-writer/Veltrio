package app

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
	if cfg.Port != 3010 {
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

func TestLoadDownloadAndLoggingSettings(t *testing.T) {
	f := filepath.Join(t.TempDir(), "config.yaml")
	content := "paths:\n  data: /tmp/vdata\n  downloads: /tmp/vdownloads\ndownloads:\n  max_concurrent: 3\n  resume_on_start: true\n  delete_file_on_remove: true\nlogging:\n  level: info\n"
	if err := os.WriteFile(f, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxConcurrent != 3 || !cfg.ResumeOnStart || !cfg.DeleteFileOnRemove || cfg.LogLevel != "info" {
		t.Fatalf("unexpected settings: %+v", cfg)
	}
}

func TestLoadNtfySettings(t *testing.T) {
	f := filepath.Join(t.TempDir(), "config.yaml")
	content := "ntfy_enabled: true\nntfy_debug: true\ndatabase_debug: true\nntfy_url: \"http://127.0.0.1:2586/topic\"\nntfy_token: \"secret\"\nntfy_notify_unknown_ipv4: true\nntfy_priority: \"high\"\nntfy_timeout_seconds: 10\n"
	if err := os.WriteFile(f, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(f)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.NtfyEnabled || !cfg.NtfyDebug || !cfg.DatabaseDebug || cfg.NtfyURL != "http://127.0.0.1:2586/topic" || cfg.NtfyToken != "secret" || !cfg.NtfyNotifyUnknownIPv4 || cfg.NtfyPriority != "high" || cfg.NtfyTimeoutSeconds != 10 {
		t.Fatalf("unexpected ntfy settings: %+v", cfg)
	}
}
