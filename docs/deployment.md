# Deployment

OpsLedger can run as a single binary or a container. SQLite is the default storage engine; PostgreSQL and MySQL can be selected with environment variables.

## Container

```bash
cp deploy/opsledger.env.example .env
docker compose up -d --build
curl http://127.0.0.1:18090/healthz
```

Open `http://127.0.0.1:18090/`. On a fresh database, the setup wizard creates the first platform administrator.

For production, edit `.env` before first start:

- Set `OPSLEDGER_CREDENTIAL_KEY` to a stable secret value.
- Keep `OPSLEDGER_DEV_SEED_USERS=0` and use the first-run setup wizard.
- Put the service behind HTTPS and set `OPSLEDGER_COOKIE_SECURE=true`.
- Store backups for the Docker volume.

## Binary With systemd

```bash
sudo ./scripts/install-systemd.sh
sudo systemctl status opsledger --no-pager
curl http://127.0.0.1:18090/healthz
```

Open the web UI and complete the first-run administrator setup if the database is empty.

The installer writes:

- Binary: `/opt/opsledger/opsledger`
- Data directory: `/var/lib/opsledger`
- Environment file: `/etc/opsledger/opsledger.env`
- Service: `/etc/systemd/system/opsledger.service`

Edit `/etc/opsledger/opsledger.env`, then restart:

```bash
sudo systemctl restart opsledger
```

## Portable Release Package

Use this mode when the target Linux or Windows server should not install Go.
The package contains the compiled executable, startup scripts, configuration example, docs, and an empty `data/` directory.

Build packages on a build machine:

```bash
./scripts/build-release.sh v0.1.0
ls -lh releases/
```

Generated files:

```text
releases/opsledger-v0.1.0-linux-amd64.tar.gz
releases/opsledger-v0.1.0-windows-amd64.zip
releases/opsledger-v0.1.0-checksums.txt
```

Run on Linux:

```bash
tar -xzf opsledger-v0.1.0-linux-amd64.tar.gz
cd opsledger-v0.1.0-linux-amd64
./start.sh
```

Run on Windows PowerShell:

```powershell
Expand-Archive .\opsledger-v0.1.0-windows-amd64.zip
cd .\opsledger-v0.1.0-windows-amd64
.\start.ps1
```

The default data path is `./data/opsledger.db` inside the extracted package.
On a fresh database, open `http://127.0.0.1:18090/` and create the first platform administrator in the setup wizard.

## PostgreSQL

Example:

```bash
OPSLEDGER_DB_DRIVER=postgres
OPSLEDGER_DB_DSN='postgres://opsledger:password@127.0.0.1:5432/opsledger?sslmode=disable'
```

## MySQL

Example:

```bash
OPSLEDGER_DB_DRIVER=mysql
OPSLEDGER_DB_DSN='opsledger:password@tcp(127.0.0.1:3306)/opsledger?parseTime=true&charset=utf8mb4'
```

## Backups

SQLite backup example:

```bash
sqlite3 /var/lib/opsledger/opsledger.db ".backup '/var/lib/opsledger/opsledger-$(date +%Y%m%d-%H%M%S).db'"
```

PostgreSQL and MySQL should use their native backup tools.

## Upgrade

Binary deployment:

```bash
sudo systemctl stop opsledger
sudo cp /opt/opsledger/opsledger /opt/opsledger/opsledger.bak.$(date +%Y%m%d-%H%M%S)
sudo ./scripts/install-systemd.sh
curl http://127.0.0.1:18090/healthz
```

Container deployment:

```bash
docker compose pull || true
docker compose up -d --build
curl http://127.0.0.1:18090/healthz
```
