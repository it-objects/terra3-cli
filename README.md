# terra3-cli

CLI for Terra3 environments.

## Commands

### Platform-authenticated (no AWS profile)

```bash
# ~/.terra3/config.yaml
api_url: https://platform.terra3.io
cognito_domain: https://login.terra3.io
cognito_client_id: <from SST / Cognito console>
identity_provider: EntraID

terra3 platform login
terra3 platform whoami
terra3 platform db-portforward                 # interactive stage/partition/port
terra3 platform db-portforward --stage <id> --partition <name> --local-port 15432 --show-password
terra3 platform logout
```

Requires the workload account's **agent stack ≥ 1.5.0**. The control plane brokers Session Manager; the laptop never holds AWS credentials for the data-plane account.

### Break-glass (AWS profile)

```bash
terra3 db port-forward -p <aws-profile>
```

Uses local AWS credentials (typically IAM Identity Center) to call `ssm:StartSession` directly. Keep this path available for emergencies until Identity Center assignments are fully withdrawn. After withdrawal, only operators with remaining account access can use it.

### Experimental AWS SSO login

```bash
terra3 login   # IAM Identity Center device flow (lists accounts; does not persist profiles)
```
