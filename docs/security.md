# Security Guide

## Do Not Publish Runtime Data

Never publish:

- `data/*.db`, `*.db-wal`, `*.db-shm`
- `.env` files
- cloud credentials
- SSH private keys
- backups
- logs that may include URLs, hostnames, usernames, or IP addresses

## Production Checklist

- Disable development seed users and use the first-run setup wizard for the initial administrator.
- Set a stable `OPSLEDGER_CREDENTIAL_KEY`.
- Serve through HTTPS.
- Set `OPSLEDGER_COOKIE_SECURE=true`.
- Use strict SSH host key verification.
- Review role permissions and approval flows before adding real credentials.
- Back up the database and credential key together.

## WebSSH

WebSSH should be treated as privileged access. Use short approval durations, strict host key verification, and audit event review.
