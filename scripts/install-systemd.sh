#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALL_DIR="${INSTALL_DIR:-/opt/opsledger}"
DATA_DIR="${DATA_DIR:-/var/lib/opsledger}"
CONFIG_DIR="${CONFIG_DIR:-/etc/opsledger}"
SERVICE_FILE="/etc/systemd/system/opsledger.service"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "Run as root, or use: sudo $0" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "Go is required to build OpsLedger. Install Go 1.22 or newer first." >&2
  exit 1
fi

if ! id opsledger >/dev/null 2>&1; then
  useradd --system --home-dir "$DATA_DIR" --create-home --shell /usr/sbin/nologin opsledger
fi

mkdir -p "$INSTALL_DIR" "$DATA_DIR" "$CONFIG_DIR"
chown opsledger:opsledger "$DATA_DIR"

tmp_bin="$(mktemp)"
(
  cd "$ROOT_DIR"
  CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o "$tmp_bin" ./cmd/opsledger
)
install -m 0755 "$tmp_bin" "$INSTALL_DIR/opsledger"
rm -f "$tmp_bin"

if [[ ! -f "$CONFIG_DIR/opsledger.env" ]]; then
  install -m 0640 "$ROOT_DIR/deploy/opsledger.env.example" "$CONFIG_DIR/opsledger.env"
  chown root:opsledger "$CONFIG_DIR/opsledger.env"
  sed -i 's#OPSLEDGER_DATA=.*#OPSLEDGER_DATA=/var/lib/opsledger/opsledger.db#' "$CONFIG_DIR/opsledger.env"
fi

install -m 0644 "$ROOT_DIR/deploy/opsledger.service" "$SERVICE_FILE"
systemctl daemon-reload
systemctl enable --now opsledger

echo "OpsLedger installed."
echo "Service: systemctl status opsledger --no-pager"
echo "Health:  curl http://127.0.0.1:18090/healthz"
