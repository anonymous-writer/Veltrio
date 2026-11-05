#!/usr/bin/env bash
set -Eeuo pipefail

# Aktualisiert Veltrio auf Raspberry Pi OS / Linux ARM64.
# Installiert nur die Binary und startet den vorhandenen systemd-User-Service neu.
# Service-Datei und Konfiguration werden nicht erstellt oder verändert.

BASE_DIR="/media/pi/Ablage/Datenablage"
ARCHIVE_NAME="veltrio-linux-arm64.tar.gz"
BINARY_NAME="veltrio-linux-arm64"
ARCHIVE_PATH="${BASE_DIR}/${ARCHIVE_NAME}"
EXTRACTED_PATH="${BASE_DIR}/${BINARY_NAME}"
SUMS_PATH="${BASE_DIR}/SHA256SUMS"

USER_NAME="${USER:-$(id -un)}"
USER_HOME="${HOME:?HOME ist nicht gesetzt}"
INSTALL_DIR="${USER_HOME}/.local/bin"
INSTALL_PATH="${INSTALL_DIR}/veltrio"
SERVICE_NAME="veltrio.service"

usage() {
  cat <<'USAGE'
Verwendung: ./update-veltrio-arm64.sh

Erwartet das Release-Archiv:
  /media/pi/Ablage/Datenablage/veltrio-linux-arm64.tar.gz

Alternativ kann eine einzelne Binary dort liegen:
  /media/pi/Ablage/Datenablage/veltrio-linux-arm64

Optional liegt SHA256SUMS im selben Verzeichnis.

Veltrio wird unter ~/.local/bin installiert und der bereits vorhandene
systemd-User-Service anschließend neu gestartet. Service-Datei und
Konfiguration bleiben vollständig unangetastet.
Das Skript bitte als normaler Benutzer (z.B. pi), nicht mit sudo, starten.
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then usage; exit 0; fi
if [[ "$#" -gt 0 ]]; then echo "Fehler: Keine Argumente erlaubt." >&2; usage >&2; exit 2; fi
if [[ "$EUID" -eq 0 ]]; then echo "Fehler: Bitte ohne sudo als normaler Benutzer ausführen." >&2; exit 1; fi
case "$(uname -m)" in aarch64|arm64) ;; *) echo "Fehler: Nur ARM64 unterstützt; erkannt: $(uname -m)" >&2; exit 1 ;; esac
if [[ "$USER_NAME" != "pi" ]]; then
  echo "Hinweis: Ausführender Benutzer ist '$USER_NAME' (nicht 'pi'). Installationspfade werden relativ zu HOME=$USER_HOME verwendet."
fi

for cmd in tar sha256sum install systemctl grep head; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "Fehler: Programm fehlt: $cmd" >&2; exit 1; }
done

# Vor der Installation prüfen, ob der User-Service bereits existiert.
# Dieses Skript legt ausdrücklich keine Service-Datei an.
if ! systemctl --user cat "$SERVICE_NAME" >/dev/null 2>&1; then
  echo "Fehler: Vorhandener User-Service '$SERVICE_NAME' nicht gefunden." >&2
  echo "Es wurde nichts installiert. Service bitte separat einrichten." >&2
  exit 1
fi

[[ -d "$BASE_DIR" ]] || { echo "Fehler: Verzeichnis fehlt: $BASE_DIR" >&2; exit 1; }
if [[ ! -f "$ARCHIVE_PATH" && ! -f "$EXTRACTED_PATH" ]]; then
  echo "Fehler: Weder Release-Archiv noch Binary gefunden." >&2
  echo "Erwartet: $ARCHIVE_PATH oder $EXTRACTED_PATH" >&2
  exit 1
fi

if [[ -f "$SUMS_PATH" ]]; then
  echo "Prüfe SHA256SUMS..."
  (cd "$BASE_DIR" && sha256sum --check --ignore-missing "$SUMS_PATH") || {
    echo "Prüfsummenprüfung fehlgeschlagen; Installation abgebrochen." >&2
    exit 1
  }
else
  echo "Warnung: SHA256SUMS nicht gefunden; keine externe Prüfsumme geprüft."
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

if [[ -f "$ARCHIVE_PATH" ]]; then
  ARCHIVE_MEMBER="$(tar -tzf "$ARCHIVE_PATH" | grep -E '(^|/)veltrio-linux-arm64$' | head -n 1 || true)"
  [[ -n "$ARCHIVE_MEMBER" ]] || {
    echo "Archiv enthält ${BINARY_NAME} nicht." >&2
    tar -tzf "$ARCHIVE_PATH" >&2 || true
    exit 1
  }
  tar -xOzf "$ARCHIVE_PATH" "$ARCHIVE_MEMBER" > "$TMP_DIR/$BINARY_NAME"

else
  echo "Kein Archiv gefunden; verwende vorhandene Binary $EXTRACTED_PATH."
  cp "$EXTRACTED_PATH" "$TMP_DIR/$BINARY_NAME"
fi
chmod 0755 "$TMP_DIR/$BINARY_NAME"

# Vorhandene Installation sichern und neue Binary atomar einsetzen.
mkdir -p "$INSTALL_DIR"
if [[ -e "$INSTALL_PATH" ]]; then
  BACKUP_PATH="${INSTALL_PATH}.bak.$(date +%Y%m%d-%H%M%S)"
  cp -a "$INSTALL_PATH" "$BACKUP_PATH"
  echo "Vorherige Binary gesichert: $BACKUP_PATH"
fi
install -m 0755 "$TMP_DIR/$BINARY_NAME" "${INSTALL_PATH}.new"
mv -f "${INSTALL_PATH}.new" "$INSTALL_PATH"

# Nur den vorhandenen Service neu starten.
# Kein Schreiben einer Unit-Datei, kein daemon-reload und kein enable.
echo "Starte vorhandenen Service '$SERVICE_NAME' neu..."
systemctl --user restart "$SERVICE_NAME"

if ! systemctl --user is-active --quiet "$SERVICE_NAME"; then
  echo "Fehler: Binary installiert, aber Service ist nicht aktiv." >&2
  systemctl --user status "$SERVICE_NAME" --no-pager >&2 || true
  exit 1
fi

echo
echo "Update abgeschlossen."
echo "Binary:        $INSTALL_PATH"
echo "Service:       $SERVICE_NAME (vorhandene Unit unverändert)"
echo "Status:        systemctl --user status $SERVICE_NAME"
echo "Logs:          journalctl --user -u $SERVICE_NAME -f"
echo "Veltrio-URL:   http://127.0.0.1:3010"
