#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  VERSION="$(git -C "$ROOT_DIR" describe --tags --always --dirty 2>/dev/null || date +%Y%m%d%H%M%S)"
fi

OUT_DIR="$ROOT_DIR/releases"
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

archive_windows() {
  local src_dir="$1"
  local out_file="$2"
  local parent
  local base
  parent="$(dirname "$src_dir")"
  base="$(basename "$src_dir")"
  if command -v zip >/dev/null 2>&1; then
    (cd "$parent" && zip -qr "$out_file" "$base")
    return
  fi
  python3 - "$parent" "$base" "$out_file" <<'PY'
from pathlib import Path
import sys
import zipfile

parent = Path(sys.argv[1])
base = sys.argv[2]
out_file = Path(sys.argv[3])
root = parent / base
with zipfile.ZipFile(out_file, "w", zipfile.ZIP_DEFLATED) as zf:
    for path in root.rglob("*"):
        if path.is_file():
            zf.write(path, path.relative_to(parent).as_posix())
PY
}

build_one() {
  local goos="$1"
  local goarch="$2"
  local ext="$3"
  local name="opsledger-${VERSION}-${goos}-${goarch}"
  local pkg_dir="$WORK_DIR/$name"

  mkdir -p "$pkg_dir/data" "$pkg_dir/config" "$pkg_dir/docs"

  echo "building $name"
  (
    cd "$ROOT_DIR"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build \
      -trimpath \
      -ldflags="-s -w" \
      -o "$pkg_dir/opsledger${ext}" \
      ./cmd/opsledger
  )

  cp "$ROOT_DIR/deploy/opsledger.env.example" "$pkg_dir/config/opsledger.env"
  sed -i \
    -e 's#^OPSLEDGER_ADDR=.*#OPSLEDGER_ADDR=127.0.0.1:18090#' \
    -e 's#^OPSLEDGER_DATA=.*#OPSLEDGER_DATA=#' \
    -e 's#^OPSLEDGER_DB_DRIVER=.*#OPSLEDGER_DB_DRIVER=sqlite3#' \
    "$pkg_dir/config/opsledger.env"
  cp "$ROOT_DIR/README.md" "$pkg_dir/docs/README.md"
  cp "$ROOT_DIR/README.zh-CN.md" "$pkg_dir/docs/README.zh-CN.md"
  cp "$ROOT_DIR/LICENSE" "$pkg_dir/LICENSE"

  cat >"$pkg_dir/README.txt" <<EOF
OpsLedger ${VERSION}

This release package is self-contained. The target machine does not need Go installed.

Linux:
  1. Edit config/opsledger.env if needed.
  2. Run ./start.sh
  3. Open http://127.0.0.1:18090/

Windows:
  1. Edit config\\opsledger.env if needed.
  2. Run .\\start.ps1 in PowerShell.
  3. Open http://127.0.0.1:18090/

Fresh databases open the first-run setup wizard. No default weak password is created.
Runtime SQLite data is stored in ./data/opsledger.db by default.
EOF

  cat >"$pkg_dir/start.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$DIR/config/opsledger.env"
if [[ -f "$ENV_FILE" ]]; then
  while IFS='=' read -r key value; do
    [[ -z "$key" || "$key" =~ ^[[:space:]]*# ]] && continue
    key="$(echo "$key" | xargs)"
    value="${value#"${value%%[![:space:]]*}"}"
    value="${value%"${value##*[![:space:]]}"}"
    if [[ -z "${!key:-}" ]]; then
      export "$key=$value"
    fi
  done < "$ENV_FILE"
fi
export OPSLEDGER_ADDR="${OPSLEDGER_ADDR:-127.0.0.1:18090}"
export OPSLEDGER_DATA="${OPSLEDGER_DATA:-$DIR/data/opsledger.db}"
export OPSLEDGER_DB_DRIVER="${OPSLEDGER_DB_DRIVER:-sqlite3}"
echo "OpsLedger is starting at http://$OPSLEDGER_ADDR"
echo "Data file: $OPSLEDGER_DATA"
exec "$DIR/opsledger" "$@"
EOF
  chmod +x "$pkg_dir/start.sh"

  cat >"$pkg_dir/start.ps1" <<'EOF'
$ErrorActionPreference = "Stop"
$Dir = Split-Path -Parent $MyInvocation.MyCommand.Path
$EnvFile = Join-Path $Dir "config\opsledger.env"
if (Test-Path $EnvFile) {
  Get-Content $EnvFile | ForEach-Object {
    $line = $_.Trim()
    if ($line -eq "" -or $line.StartsWith("#")) { return }
    $idx = $line.IndexOf("=")
    if ($idx -le 0) { return }
    $key = $line.Substring(0, $idx).Trim()
    $value = $line.Substring($idx + 1).Trim()
    if (-not [Environment]::GetEnvironmentVariable($key, "Process")) {
      [Environment]::SetEnvironmentVariable($key, $value, "Process")
    }
  }
}
if (-not $env:OPSLEDGER_ADDR) { $env:OPSLEDGER_ADDR = "127.0.0.1:18090" }
if (-not $env:OPSLEDGER_DATA) { $env:OPSLEDGER_DATA = Join-Path $Dir "data\opsledger.db" }
if (-not $env:OPSLEDGER_DB_DRIVER) { $env:OPSLEDGER_DB_DRIVER = "sqlite3" }
Write-Host "OpsLedger is starting at http://$env:OPSLEDGER_ADDR"
Write-Host "Data file: $env:OPSLEDGER_DATA"
& (Join-Path $Dir "opsledger.exe") @args
EOF

  if [[ "$goos" == "windows" ]]; then
    archive_windows "$pkg_dir" "$OUT_DIR/$name.zip"
  else
    tar -C "$WORK_DIR" -czf "$OUT_DIR/$name.tar.gz" "$name"
  fi
}

mkdir -p "$OUT_DIR"
build_one linux amd64 ""
build_one windows amd64 ".exe"

(
  cd "$OUT_DIR"
  sha256sum \
    "opsledger-$VERSION-linux-amd64.tar.gz" \
    "opsledger-$VERSION-windows-amd64.zip" \
    > "opsledger-$VERSION-checksums.txt"
)

echo "Release files:"
ls -lh "$OUT_DIR"/opsledger-"$VERSION"-*
