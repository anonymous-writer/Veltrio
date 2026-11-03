# Veltrio

Eigenständiger Linux Download Manager mit Go-Backend und responsiver Weboberfläche im LeaseEditor-inspirierten Look.

## Funktionen
- HTTP/HTTPS-Downloads
- Hinzufügen, pausieren, fortsetzen und löschen
- Fortschritt, Dateigröße und Geschwindigkeit
- Persistenter Downloadstatus und Fortsetzen nach Neustart (wenn der Server Range-Requests unterstützt)
- Docker Multi-Arch: Linux AMD64 und ARM64
- Native Linux-Binaries: AMD64 und ARM64

## Docker
```bash
docker compose up -d --build
```
Danach `http://localhost:9999` öffnen. Downloads liegen in `./Downloads`, Statusdaten in `./data`.

## Native Builds
```bash
go test ./...
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o veltrio-linux-amd64 ./cmd/veltrio
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o veltrio-linux-arm64 ./cmd/veltrio
```

## Konfiguration
- `VELTRIO_ADDR` (Standard `:9999`)
- `VELTRIO_DATA` (Standard `/app/data`)
- `VELTRIO_DOWNLOADS` (Standard `/app/Downloads`)

## Hinweis
Dies ist eine bewusst schlanke erste Eigenentwicklung. Aktuell werden HTTP und HTTPS unterstützt; FTP, BitTorrent, Browser-Erweiterungen und parallele Segment-Downloads sind nicht enthalten.
