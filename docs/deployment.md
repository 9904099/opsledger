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
