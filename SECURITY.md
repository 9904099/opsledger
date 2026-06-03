# Security Policy

OpsLedger manages cloud accounts, credentials, infrastructure assets, approvals, and WebSSH sessions. Treat every deployment as sensitive.

## Do Not Commit

Never commit:

- SQLite databases, WAL/SHM files, backups, dumps, or exported ledgers.
- Cloud credentials, API tokens, SSH keys, kubeconfigs, or known_hosts files.
- Real host inventories, private domains, internal IPs, deployment logs, or incident evidence.
- Release packages that bundle databases or environment files.

The project `.gitignore` excludes the common local paths, but review `git status` before every commit.

## Development Defaults

Development seed users are only created when `OPSLEDGER_DEV_SEED_USERS=1` is set. Do not enable this in production. Replace local passwords or connect SSO/OIDC before exposing the service.

## Production Notes

- Set `OPSLEDGER_CREDENTIAL_KEY` to a strong random value before storing credentials.
- Use HTTPS and set `OPSLEDGER_COOKIE_SECURE=true`.
- Configure strict SSH host key checking for WebSSH and PVE discovery.
- Keep the database encrypted or protected by host-level access controls.

## Reporting

For security issues, open a private security advisory or contact the repository owner through GitHub.
