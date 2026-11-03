package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"veltrio/internal/server"
)

func main() {
	dataDir := env("VELTRIO_DATA", "/app/data")
	downloadDir := env("VELTRIO_DOWNLOADS", "/app/Downloads")
	addr := env("VELTRIO_ADDR", ":9999")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		log.Fatal(err)
	}
	app, err := server.New(filepath.Join(dataDir, "veltrio.json"), downloadDir)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Veltrio listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, app.Handler()))
}
func env(k, f string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return f
}
