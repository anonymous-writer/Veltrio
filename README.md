# Veltrio

**Ein einfacher Downloadmanager für Linux.**

Veltrio basiert auf der Download-Engine von [Gopeed](https://github.com/GopeedLab/gopeed) und verwendet die Gestaltung deines LeaseEditor-Projekts (dunkelblau, Blau/Cyan-Akzente und dein Logo). Die Oberfläche konzentriert sich auf die Grundfunktionen: URL einfügen, Download starten, pausieren, fortsetzen, entfernen und Fortschritt ansehen.

## Unterstützte Ziele

| Variante | Architektur |
|---|---|
| Docker Multi-Arch | Linux AMD64 (`linux/amd64`) |
| Docker Multi-Arch | Linux ARM64 (`linux/arm64`) |
| Native Linux-Binärdatei | AMD64 |
| Native Linux-Binärdatei | ARM64 |

Es werden keine Windows-, macOS-, Android- oder iOS-Builds und keine DEB-, RPM-, PKG- oder sonstigen Installationspakete erzeugt.

## Docker

Im Verzeichnis mit `compose.yaml` und `config.json`:

```bash
mkdir -p data Downloads
docker compose up -d
```

Das Compose-Beispiel verwendet `ghcr.io/DEIN-GITHUB-NAME/veltrio:latest`. Ersetze den Platzhalter nach dem Anlegen deines GitHub-Repositories durch dessen Namen.

Die Weboberfläche ist anschließend unter `http://SERVER-IP:9999` erreichbar.

Die Container-Verzeichnisse:
- `/app/data` – interne Downloadmanager-Datenbank und Einstellungen
- `/app/Downloads` – heruntergeladene Dateien

Der Container läuft als unprivilegierter Benutzer. Stelle sicher, dass die eingebundenen Host-Verzeichnisse für die Container-UID schreibbar sind.

## Native Linux-Version

Die passenden Binärdateien `veltrio-linux-amd64` oder `veltrio-linux-arm64` aus den GitHub-Releases herunterladen. Daneben `config.json` ablegen und aus diesem Verzeichnis starten:

```bash
chmod +x veltrio-linux-amd64
./veltrio-linux-amd64 --config ./config.json
```

Für ARM64 entsprechend `veltrio-linux-arm64` verwenden. Die Downloads werden standardmäßig im Unterordner `Downloads` abgelegt; die Datenbank liegt in `data`.

## Konfiguration

`config.json` ist für Docker und native Starts gleich aufgebaut:

```json
{
  "address": "0.0.0.0",
  "port": 9999,
  "storageDir": "./data",
  "downloadConfig": {
    "downloadDir": "./Downloads"
  }
}
```

Optional lassen sich `username` und `password` in `config.json` ergänzen, um die Webanmeldung zu aktivieren. Wenn du Veltrio im Netzwerk erreichbar machst, setze ein eigenes Passwort und veröffentliche den Port nicht ungeschützt im Internet.

## Oberfläche

Enthalten sind nur:
- Download per URL hinzufügen
- Download pausieren und fortsetzen
- Download entfernen
- Status, Fortschritt, Dateigröße und Geschwindigkeit anzeigen

Keine Add-on-Verwaltung, keine Erweiterungsoberfläche, keine Desktop-Clients, kein Updater und keine Paket-Installer.

## Build

Die GitHub-Actions-Workflows bauen ausschließlich:
- Docker Multi-Arch: `linux/amd64` und `linux/arm64`
- Native Linux: `amd64` und `arm64`

Native Builds werden als einzelne ausführbare Dateien veröffentlicht. Es werden keine Paketformate erstellt.

## Lizenz und Herkunft

Veltrio verwendet den Gopeed-Quellcode als technische Download-Engine. Der Upstream-Quellcode ist unter GPL-3.0 lizenziert; diese Lizenz- und Weitergabepflichten gelten auch für abgeleitete Versionen. Die originale Lizenzdatei bleibt im Repository enthalten. Das Veltrio-Branding und die Oberfläche wurden für dieses Projekt angepasst.
