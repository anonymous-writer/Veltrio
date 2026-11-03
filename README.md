# Veltrio

Eigenständiger Linux Download Manager mit Go-Backend und responsiver Weboberfläche im LeaseEditor-inspirierten Look.

## Funktionen
- HTTP/HTTPS-Downloads
- Hinzufügen, pausieren, fortsetzen und löschen
- Fortschritt, Dateigröße und Geschwindigkeit
- Persistenter Downloadstatus und Fortsetzen nach Neustart (wenn der Server Range-Requests unterstützt)
- Docker Multi-Arch: Linux AMD64 und ARM64
- Native Linux-Binaries: AMD64 und ARM64
- YAML-Konfiguration mit Startparametern zum Überschreiben

## Native Installation und Start
```bash
./veltrio
```
Beim ersten Start legt Veltrio automatisch `config.yaml` im aktuellen Verzeichnis an. Die Standardpfade für Daten und Downloads liegen im Home-Verzeichnis des ausführenden Benutzers.

## config.yaml
```yaml
server:
  host: "0.0.0.0"
  port: 9999

paths:
  data: "~/.local/share/veltrio"
  downloads: "~/Downloads/Veltrio"
```
`~` wird zum Home-Verzeichnis des ausführenden Benutzers aufgelöst. Umgebungsvariablen wie `$HOME` werden ebenfalls expandiert.

## Startparameter
```bash
veltrio --help
veltrio --version
veltrio --config /pfad/zu/config.yaml
veltrio --port 8080
veltrio --host 127.0.0.1
veltrio --data /pfad/zu/daten
veltrio --downloads /pfad/zu/downloads
```
Explizite Startparameter überschreiben die Konfiguration. Danach folgen Umgebungsvariablen und zuletzt die YAML-Datei (Priorität: CLI > Environment > YAML).

## Docker
```bash
docker compose up -d --build
```
Danach `http://localhost:9999` öffnen. Downloads liegen in `./Downloads`, Statusdaten und automatisch angelegte Konfiguration in `./data`.

## Native Builds
```bash
go test ./...
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o veltrio-linux-amd64 ./cmd/veltrio
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o veltrio-linux-arm64 ./cmd/veltrio
```

## Hinweis
Dies ist eine bewusst schlanke erste Eigenentwicklung. Aktuell werden HTTP und HTTPS unterstützt; FTP, BitTorrent, Browser-Erweiterungen und parallele Segment-Downloads sind nicht enthalten.
