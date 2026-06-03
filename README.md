# OpsLedger

OpsLedger is a lightweight cloud operations ledger for small platform and SRE teams. It tracks cloud accounts, assets, credentials, approvals, audit events, probes, cost snapshots, and controlled WebSSH access in a single Go service.

## Features

- Single Go binary with embedded web UI.
- SQLite by default, with PostgreSQL and MySQL DSN support.
- Cloud account and asset ledger for AWS, Cloudflare, PVE, Aliyun, Tencent Cloud, and manual assets.
- AWS discovery for common resource types and Cost Explorer snapshots.
- Cloudflare discovery for zones, DNS records, Workers, R2 buckets, WAF rulesets, and load balancers.
- PVE discovery through read-only SSH commands.
- Local login, role-based workspaces, approval flows, audit events, and credential encryption.
- Optional WebSSH temporary access flow for EC2 assets.
- Container and systemd binary deployment examples.

## Quick Start

Run with a local SQLite database:

```bash
go run ./cmd/opsledger
```

Open `http://127.0.0.1:18090/`.

On first deployment, OpsLedger initializes the database schema automatically. If no user exists, the login page switches to the setup wizard and asks you to create the first platform administrator. No default weak password is created.

## Container Deployment

```bash
cp deploy/opsledger.env.example .env
docker compose up -d --build
```

The service listens on `http://localhost:18090/` and stores SQLite data in the `opsledger-data` Docker volume.

## Binary Deployment

On a Linux host with Go installed:

```bash
sudo ./scripts/install-systemd.sh
sudo systemctl status opsledger --no-pager
```

The installer builds `./cmd/opsledger`, installs it under `/opt/opsledger`, creates `/var/lib/opsledger`, writes `/etc/opsledger/opsledger.env` if missing, and enables the `opsledger` systemd service.

## Important Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `OPSLEDGER_ADDR` | `127.0.0.1:18090` | HTTP listen address. Use `0.0.0.0:18090` in containers. |
| `OPSLEDGER_DATA` | `data/opsledger.db` | SQLite database path. |
| `OPSLEDGER_DB_DRIVER` | `sqlite3` | `sqlite3`, `postgres`, or `mysql`. |
| `OPSLEDGER_DB_DSN` | empty | DSN for the selected database driver. |
| `OPSLEDGER_CREDENTIAL_KEY` | derived dev key | 32-byte base64 or plain key for credential encryption. Set this in production. |
| `OPSLEDGER_COOKIE_SECURE` | `false` | Set to `true` behind HTTPS. |
| `OPSLEDGER_DEV_SEED_USERS` | `false` | Creates local test users only when enabled. |
| `OPSLEDGER_DEV_SEED_PASSWORD` | empty | Password used by dev seed users. |
| `OPSLEDGER_SEED_EXAMPLE_TOOLS` | `false` | Optionally creates example tool entries. |
| `OPSLEDGER_SSH_STRICT_HOST_KEY` | `false` | Enforce SSH host key checking for WebSSH. |
| `OPSLEDGER_SSH_KNOWN_HOSTS` | empty | known_hosts file for WebSSH. |
| `OPSLEDGER_PVE_SSH_STRICT_HOST_KEY` | `false` | Enforce SSH host key checking for PVE discovery. |
| `OPSLEDGER_PVE_SSH_KNOWN_HOSTS` | empty | known_hosts file for PVE discovery. |

## Repository Layout

```text
cmd/opsledger/          Service entrypoint
internal/app/           HTTP API, embedded UI, auth, approvals, WebSSH
internal/discovery/     AWS, Cloudflare, PVE discovery
internal/model/         Data models
internal/store/         Storage, migrations, seed data, RBAC, audit, credentials
deploy/                 systemd and env examples
scripts/                install and operations scripts
docs/                   Public documentation
data/.gitkeep           Placeholder only; runtime DB files are ignored
```

## Security Notes

- Do not commit runtime databases, `.env` files, backups, private keys, or cloud credentials.
- Set `OPSLEDGER_CREDENTIAL_KEY` before storing real credentials.
- Use HTTPS and set `OPSLEDGER_COOKIE_SECURE=true` in production.
- Use strict SSH host key verification for production WebSSH/PVE usage.
- Development seed users must be disabled for public or production deployments.

See [SECURITY.md](./SECURITY.md) and [docs/deployment.md](./docs/deployment.md).

## License

MIT
