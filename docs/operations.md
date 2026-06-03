# Operations

## Health Check

```bash
curl http://127.0.0.1:18090/healthz
```

Expected response:

```text
ok
```

## Logs

systemd:

```bash
journalctl -u opsledger -f
```

Docker Compose:

```bash
docker compose logs -f opsledger
```

## SQLite Backup

```bash
mkdir -p backups
sqlite3 data/opsledger.db ".backup 'backups/opsledger-$(date +%Y%m%d-%H%M%S).db'"
```

## SQLite Restore

Stop the service first, then replace the database with a verified backup.

```bash
systemctl stop opsledger
cp /var/lib/opsledger/opsledger.db /var/lib/opsledger/opsledger.db.before-restore
cp /path/to/backup.db /var/lib/opsledger/opsledger.db
chown opsledger:opsledger /var/lib/opsledger/opsledger.db
systemctl start opsledger
curl http://127.0.0.1:18090/healthz
```

## Verification After Changes

- Login with an expected role.
- Confirm the role opens the correct workspace.
- Check `/healthz`.
- Review recent audit events.
- Confirm no unexpected cloud sync or WebSSH errors in logs.
