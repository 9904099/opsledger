# Product Notes

OpsLedger is designed around two daily workflows.

## Operations Workspace

Operations users manage:

- Cloud platforms and cloud accounts.
- Assets discovered from providers or entered manually.
- Cost snapshots and chargeback estimates.
- Inspections, probes, alerts, and changes.
- Credentials, approval flows, and audit events.

## Developer Workspace

Developers use:

- Environment application entries.
- Global tool entries.
- Credential and WebSSH access requests.
- Temporary approved access.

Developers do not need direct access to cloud account configuration.

## Auditor Workspace

Auditors review:

- Login events.
- Permission denials.
- Credential reveal and copy events.
- Approval decisions.
- WebSSH session events.
- Cloud sync and cost sync events.

## Default Roles

| Role | Purpose |
| --- | --- |
| `admin` | System configuration and high-risk approvals. |
| `ops` | Cloud accounts, assets, credentials, tools, and approvals. |
| `lead` | Team approval decisions for development and test environments. |
| `developer` | Tool, credential, and WebSSH usage requests. |
| `viewer` | Read-only ledger view. |
| `auditor` | Audit workspace access. |
