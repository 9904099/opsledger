# Requirements

| ID | Requirement | User Story | Priority | Acceptance |
| --- | --- | --- | --- | --- |
| R01 | Cloud account management | As an ops engineer, I can register cloud accounts and keep credentials encrypted. | P0 | Supports account metadata, masked credentials, manual sync, and sync history. |
| R02 | Asset ledger | As an ops engineer, I can view assets by platform, account, project, and resource type. | P0 | Assets can be discovered or manually managed; tree views support account and project grouping. |
| R03 | AWS discovery | As an ops engineer, I can import AWS resources with access keys. | P0 | Supports common AWS resources and Cost Explorer snapshots when permissions are available. |
| R04 | Cloudflare discovery | As an ops engineer, I can import zones, DNS, and edge resources. | P0 | Partial API permission failures are recorded as warnings and do not block other resource sync. |
| R05 | PVE discovery | As an ops engineer, I can register PVE as a cloud account and discover VMs. | P1 | Uses read-only SSH commands and records sync warnings. |
| R06 | Approval flow | As a developer, I can request credential or WebSSH access. | P0 | Requests follow configured approval steps and generate temporary access grants after approval. |
| R07 | Credential governance | As an ops engineer, I can store and reveal credentials with audit records. | P0 | Secrets are encrypted, masked in lists, and reveal/copy actions are audited. |
| R08 | WebSSH access | As a developer, I can open an approved temporary WebSSH session. | P1 | Session is tied to user, asset, and grant; session lifecycle is audited. |
| R09 | Audit workspace | As an auditor, I can review operational events. | P0 | Auditor has a dedicated workspace and does not enter configuration pages by default. |
| R10 | Deployment | As an operator, I can deploy with a binary or container. | P0 | Provides Docker Compose and systemd deployment paths. |
