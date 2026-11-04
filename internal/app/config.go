package app

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Host               string
	Port               int
	DataDir            string
	DownloadsDir       string
	MaxConcurrent      int
	ResumeOnStart      bool
	DeleteFileOnRemove bool
	LogLevel           string
}

func Defaults() Config {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "."
	}
	return Config{
		Host:          "0.0.0.0",
		Port:          3010,
		DataDir:       filepath.Join(home, ".local", "share", "veltrio"),
		DownloadsDir:  filepath.Join(home, "Downloads", "Veltrio"),
		MaxConcurrent: 3, ResumeOnStart: true, DeleteFileOnRemove: true, LogLevel: "info",
	}
}

// Load creates a commented starter config when the requested file does not exist.
func Load(filename string) (Config, error) {
	cfg := Defaults()
	f, err := os.Open(filename)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			return cfg, err
		}
		if err := os.WriteFile(filename, []byte(DefaultYAML(cfg)), 0644); err != nil {
			return cfg, err
		}
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	defer f.Close()

	section := ""
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" || strings.HasPrefix(line, "---") {
			continue
		}
		if !strings.Contains(line, ":") {
			return cfg, fmt.Errorf("%s:%d: erwartetes YAML-Schlüssel: Wert-Paar", filename, lineNo)
		}
		key, value, _ := strings.Cut(line, ":")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if value == "" {
			section = key
			continue
		}
		value = unquote(value)
		switch section + "." + key {
		case "server.host":
			cfg.Host = value
		case "server.port":
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 || n > 65535 {
				return cfg, fmt.Errorf("%s:%d: ungültiger server.port", filename, lineNo)
			}
			cfg.Port = n
		case "paths.data":
			cfg.DataDir = value
		case "paths.downloads":
			cfg.DownloadsDir = value
		case "downloads.max_concurrent":
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 || n > 64 {
				return cfg, fmt.Errorf("%s:%d: downloads.max_concurrent muss zwischen 1 und 64 liegen", filename, lineNo)
			}
			cfg.MaxConcurrent = n
		case "downloads.resume_on_start":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return cfg, fmt.Errorf("%s:%d: ungültiger downloads.resume_on_start", filename, lineNo)
			}
			cfg.ResumeOnStart = b
		case "downloads.delete_file_on_remove":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return cfg, fmt.Errorf("%s:%d: ungültiger downloads.delete_file_on_remove", filename, lineNo)
			}
			cfg.DeleteFileOnRemove = b
		case "logging.level":
			level := strings.ToLower(value)
			if level != "debug" && level != "info" && level != "warn" && level != "error" {
				return cfg, fmt.Errorf("%s:%d: logging.level muss debug, info, warn oder error sein", filename, lineNo)
			}
			cfg.LogLevel = level
		default:
			return cfg, fmt.Errorf("%s:%d: unbekannte Einstellung %s.%s", filename, lineNo, section, key)
		}
	}
	if err := scanner.Err(); err != nil {
		return cfg, err
	}
	if cfg.Host == "" {
		return cfg, fmt.Errorf("server.host darf nicht leer sein")
	}
	if cfg.DataDir == "" || cfg.DownloadsDir == "" {
		return cfg, fmt.Errorf("paths.data und paths.downloads dürfen nicht leer sein")
	}
	return cfg, nil
}
func DefaultYAML(c Config) string {
	return fmt.Sprintf("# Veltrio configuration\n# Command-line options override these values.\n\nserver:\n  host: %q\n  port: %d\n\npaths:\n  data: %q\n  downloads: %q\n\ndownloads:\n  max_concurrent: %d\n  resume_on_start: %t\n  delete_file_on_remove: %t\n\nlogging:\n  level: %q\n", c.Host, c.Port, c.DataDir, c.DownloadsDir, c.MaxConcurrent, c.ResumeOnStart, c.DeleteFileOnRemove, c.LogLevel)
}
func ExpandPath(p string) string {
	p = os.ExpandEnv(p)
	if p == "~" {
		if h, e := os.UserHomeDir(); e == nil {
			return h
		}
	}
	if strings.HasPrefix(p, "~/") {
		if h, e := os.UserHomeDir(); e == nil {
			return filepath.Join(h, p[2:])
		}
	}
	return filepath.Clean(p)
}
func stripComment(s string) string {
	inSingle, inDouble, escaped := false, false, false
	for i, r := range s {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && inDouble {
			escaped = true
			continue
		}
		if r == '\'' && !inDouble {
			inSingle = !inSingle
		}
		if r == '"' && !inSingle {
			inDouble = !inDouble
		}
		if r == '#' && !inSingle && !inDouble {
			return strings.TrimSpace(s[:i])
		}
	}
	return s
}
func unquote(s string) string {
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1]
	}
	return s
}
