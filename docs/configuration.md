# Configuration

OpsLedger uses environment variables for runtime configuration.

## Database

- `OPSLEDGER_DATA`: SQLite database path.
- `OPSLEDGER_DB_DRIVER`: `sqlite3`, `postgres`, or `mysql`.
- `OPSLEDGER_DB_DSN`: DSN for PostgreSQL or MySQL.

If `OPSLEDGER_DB_DRIVER` is empty, SQLite is used.

## Credentials

Set `OPSLEDGER_CREDENTIAL_KEY` before storing real credentials. Keep this value stable; changing it can make existing encrypted credentials unreadable.

## Login And First Setup

On a fresh database, use the first-run setup wizard to create the first platform administrator.

Development seed users are still available for local automated tests and demos:

```bash
OPSLEDGER_DEV_SEED_USERS=1
OPSLEDGER_DEV_SEED_PASSWORD=change-this-password
```

Disable seed users in production.

## Discovery Credentials

Cloud provider credentials are entered through the UI and encrypted before storage. Avoid storing credentials in environment files.

## SSH

For production WebSSH or PVE discovery, enable strict host key verification:

```bash
OPSLEDGER_SSH_STRICT_HOST_KEY=true
OPSLEDGER_SSH_KNOWN_HOSTS=/etc/opsledger/known_hosts
OPSLEDGER_PVE_SSH_STRICT_HOST_KEY=true
OPSLEDGER_PVE_SSH_KNOWN_HOSTS=/etc/opsledger/pve_known_hosts
```
