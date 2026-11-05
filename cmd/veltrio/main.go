package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"veltrio/internal/app"
)

var version = "dev"

func main() {
	flags := flag.NewFlagSet("veltrio", flag.ExitOnError)
	configPath := flags.String("config", env("VELTRIO_CONFIG", "config.yaml"), "Pfad zur config.yaml")
	host := flags.String("host", "", "Webserver-Adresse überschreiben")
	port := flags.Int("port", 0, "Webserver-Port überschreiben")
	dataDir := flags.String("data", "", "Datenverzeichnis überschreiben")
	downloadsDir := flags.String("downloads", "", "Download-Verzeichnis überschreiben")
	showVersion := flags.Bool("version", false, "Version anzeigen")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Veltrio %s – Linux Download Manager\n\n", version)
		fmt.Fprintf(flags.Output(), "Verwendung: veltrio [Optionen]\n\nOptionen:\n")
		flags.PrintDefaults()
		fmt.Fprintln(flags.Output(), "\nOhne vorhandene Konfiguration wird config.yaml automatisch angelegt.")
	}
	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if *showVersion {
		fmt.Println("Veltrio", version)
		return
	}

	cfg, err := app.Load(configPathExpanded(*configPath))
	if err != nil {
		log.Fatalf("Konfiguration: %v", err)
	}

	// Environment variables preserve compatibility with Docker deployments.
	if v := os.Getenv("VELTRIO_ADDR"); v != "" {
		h, p, e := net.SplitHostPort(v)
		if e == nil {
			cfg.Host = h
			if n, e := strconv.Atoi(p); e == nil {
				cfg.Port = n
			}
		} else if v[0] == ':' {
			if n, e := strconv.Atoi(v[1:]); e == nil {
				cfg.Port = n
			}
		}
	}
	if v := os.Getenv("VELTRIO_DATA"); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv("VELTRIO_DOWNLOADS"); v != "" {
		cfg.DownloadsDir = v
	}

	// Explicit command-line flags take precedence over config and environment.
	flags.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "host":
			cfg.Host = *host
		case "port":
			cfg.Port = *port
		case "data":
			cfg.DataDir = *dataDir
		case "downloads":
			cfg.DownloadsDir = *downloadsDir
		}
	})
	cfg.DataDir = app.ExpandPath(cfg.DataDir)
	cfg.DownloadsDir = app.ExpandPath(cfg.DownloadsDir)
	if cfg.Port < 1 || cfg.Port > 65535 {
		log.Fatal("Port muss zwischen 1 und 65535 liegen")
	}
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		log.Fatalf("Datenverzeichnis: %v", err)
	}
	if err := os.MkdirAll(cfg.DownloadsDir, 0755); err != nil {
		log.Fatalf("Download-Verzeichnis: %v", err)
	}

	serverApp, err := app.NewWithConfig(filepath.Join(cfg.DataDir, "veltrio.json"), cfg.DownloadsDir, version, cfg.MaxConcurrent, cfg.ResumeOnStart, cfg.DeleteFileOnRemove, cfg.LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	serverApp.ConfigureNtfy(app.Ntfy{Enabled: cfg.NtfyEnabled, Debug: cfg.NtfyDebug, URL: cfg.NtfyURL, Token: cfg.NtfyToken, Priority: cfg.NtfyPriority, Timeout: time.Duration(cfg.NtfyTimeoutSeconds) * time.Second}, cfg.DatabaseDebug, cfg.NtfyNotifyUnknownIPv4)
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	if cfg.LogLevel == "debug" || cfg.LogLevel == "info" {
		log.Printf("Veltrio %s listening on %s (config: %s, downloads: %s)", version, addr, *configPath, cfg.DownloadsDir)
	}
	log.Fatal(http.ListenAndServe(addr, serverApp.Handler()))
}
func configPathExpanded(p string) string { return app.ExpandPath(p) }
func env(k, f string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return f
}
