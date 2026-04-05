# Email Provider Configuration Guide

Configure msgvault to archive email from Gmail, Microsoft 365/Outlook, Outlook.com, and any IMAP-compatible server.

## Supported Providers

| Provider | Command | Auth Method | Incremental Sync |
|----------|---------|-------------|-----------------|
| Gmail (personal & Workspace) | `add-account` | OAuth2 (browser) | Yes (History API) |
| Microsoft 365 / Outlook.com | `add-o365` | OAuth2 + XOAUTH2 | No (full sync only) |
| Generic IMAP | `add-imap` | Username/Password | No (full sync only) |
| MBOX import | `import-mbox` | Offline file | N/A |
| Apple Mail (.emlx) | `import-emlx` | Offline file | N/A |

---

## Gmail

### Prerequisites

You need a Google Cloud OAuth "Desktop application" credential.

1. Go to [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
2. Select or create a project
3. Enable the **Gmail API** (APIs & Services > Library > search "Gmail API")
4. Go to **Credentials** > **Create Credentials** > **OAuth client ID**
5. Application type: **Desktop application**
6. Give it a name (e.g., "msgvault")
7. Download the JSON file (`client_secret_XXXXX.json`)

> **Important:** Create a **Desktop application** or **Web application** client. TV/device OAuth clients are **not supported** — Google's device flow does not support Gmail scopes.

### Configuration

Add your OAuth client secrets to `~/.msgvault/config.toml`:

```toml
[oauth]
client_secrets = "/path/to/client_secret.json"
```

**Windows paths** must use forward slashes or single-quoted backslashes:
```toml
[oauth]
client_secrets = "C:/Users/You/Downloads/client_secret.json"
# or
[oauth]
client_secrets = 'C:\Users\You\Downloads\client_secret.json'
```

### Adding an Account

```bash
# Browser-based OAuth (default)
msgvault add-account you@gmail.com

# Force re-authorization (deletes existing token first)
msgvault add-account you@gmail.com --force

# With a custom display name
msgvault add-account you@gmail.com --display-name "Work Account"
```

This opens your browser to Google's authorization page. Select the account and grant permissions. msgvault requests:
- `gmail.readonly` — Read messages, labels, and metadata
- `gmail.modify` — Modify labels (for deletion staging)

### Headless Servers

Google does not support Gmail scopes in device/TV flow. For headless servers:

```bash
msgvault add-account you@gmail.com --headless
```

This prints instructions to:
1. Authorize on a machine with a browser
2. Copy the token file (`~/.msgvault/tokens/you@gmail.com.json`) to your server via `scp`
3. Re-run `add-account` on the server — it detects the copied token and registers the account

### Google Workspace (Multiple OAuth Apps)

If your organization requires its own OAuth app:

```toml
[oauth]
client_secrets = "/path/to/personal_secret.json"

[oauth.apps.acme]
client_secrets = "/path/to/acme_workspace_secret.json"

[oauth.apps.contoso]
client_secrets = "/path/to/contoso_secret.json"
```

```bash
# Use a named OAuth app
msgvault add-account you@acme.com --oauth-app acme

# Personal Gmail uses the default
msgvault add-account personal@gmail.com
```

### Syncing

```bash
# Full sync (read-only)
msgvault sync-full you@gmail.com

# Incremental sync (only new/changed messages)
msgvault sync you@gmail.com

# Sync a date range
msgvault sync-full you@gmail.com --after 2024-01-01 --before 2024-12-31

# Sync with a Gmail search query
msgvault sync-full you@gmail.com --query "from:boss@company.com"

# Limit for testing
msgvault sync-full you@gmail.com --limit 100
```

### Deletion Scopes

To permanently delete messages from Gmail (not just trash), you need the full `https://mail.google.com/` scope. If your token was authorized with only `gmail.modify`, re-authorize with the deletion scope:

```bash
msgvault add-account you@gmail.com --force
```

msgvault tracks which scopes each token was authorized with and will prompt you when a wider scope is needed.

### Troubleshooting

| Issue | Solution |
|-------|----------|
| "OAuth client secrets not configured" | Add `[oauth] client_secrets` to config.toml |
| "TV/device clients are not supported" | Create a Desktop application, not a TV/device client |
| "Token mismatch" | You selected the wrong Google account — re-run with the correct email |
| "invalid_grant" | Token expired; msgvault auto-reauthorizes during sync |
| Port 8089 already in use | Another process is using the callback port — stop it and retry |

---

## Microsoft 365 / Outlook.com

msgvault supports both **organizational** Microsoft 365 accounts (company/school) and **personal** Microsoft accounts (outlook.com, hotmail.com, live.com, msn.com).

Authentication uses OAuth2 with XOAUTH2 over IMAP — no Microsoft Graph API is required.

### Prerequisites: Azure AD App Registration

1. Go to the [Azure Portal](https://portal.azure.com) > **Microsoft Entra ID** > **App registrations**
2. Click **New registration**
3. **Name:** `msgvault` (or any name you prefer)
4. **Supported account types:**
   - **Personal Microsoft accounts** (outlook.com, hotmail.com, etc.): Select "Accounts in any organizational directory and personal Microsoft accounts"
   - **Organizational accounts only** (company/school): Select "Accounts in any organizational directory" (multi-tenant) or "Accounts in this organizational directory only" (single-tenant)
5. **Redirect URI:**
   - Platform: **Web**
   - URL: `http://localhost`
6. Enable both **ID tokens** and **Access tokens** in the authentication settings
7. Click **Register**
8. Copy the **Application (client) ID** — this is your `client_id`

> **Tenant ID:** For most setups, leave `tenant_id` unset (defaults to `"common"` for multi-tenant). For single-tenant org setups, find your tenant ID in the Azure Portal under **Microsoft Entra ID** > **Overview** > **Tenant ID**.

### Configuration

Add your Azure AD app to `~/.msgvault/config.toml`:

```toml
[microsoft]
client_id = "your-azure-app-client-id"
tenant_id = "common"   # optional; omit for multi-tenant (default)
```

### Adding an Account

```bash
# Personal account (outlook.com, hotmail.com, live.com, msn.com)
msgvault add-o365 you@outlook.com

# Organizational account (Microsoft 365 / company)
msgvault add-o365 you@company.com

# With explicit tenant ID (overrides config.toml)
msgvault add-o365 you@company.com --tenant your-tenant-id
```

This opens your browser to Microsoft's authorization page. Sign in and grant permissions. msgvault requests:
- **Personal accounts:** `https://outlook.office.com/IMAP.AccessAsUser.All`, `offline_access`, `openid`, `email`
- **Organizational accounts:** `https://outlook.office365.com/IMAP.AccessAsUser.All`, `offline_access`, `openid`, `email`

msgvault automatically detects your account type and selects the correct IMAP scope and server:

| Account Type | IMAP Scope | IMAP Host |
|-------------|------------|-----------|
| Personal (outlook.com, hotmail.com, etc.) | `outlook.office.com/IMAP.AccessAsUser.All` | `outlook.office.com:993` |
| Organizational (company/school) | `outlook.office365.com/IMAP.AccessAsUser.All` | `outlook.office365.com:993` |

### IMAP Scope Correction

If you authorize with the wrong IMAP scope (e.g., personal scope for an org account), msgvault detects the mismatch from the tenant ID (`tid` claim) in the ID token and automatically re-authorizes with the correct scope. You'll see a log message like:

```
correcting IMAP scope based on tenant ID, re-authorizing
```

### Syncing

```bash
# Full sync (read-only)
msgvault sync-full you@company.com

# Sync a date range
msgvault sync-full you@company.com --after 2024-01-01

# Note: Microsoft 365 does not support incremental sync
# Each sync-full performs a complete scan via IMAP
```

### Re-authorization

If you need to re-authorize (e.g., after scope correction or token revocation):

```bash
msgvault add-o365 you@company.com
```

msgvault detects the existing Microsoft XOAUTH2 source and updates it in place rather than creating a duplicate.

### Removing an Account

```bash
msgvault remove-account you@company.com
```

This revokes the refresh token at Microsoft (best-effort) and removes the local token file. Revoked tokens expire naturally within 90 days if revocation fails.

### Troubleshooting

| Issue | Solution |
|-------|----------|
| "Microsoft OAuth not configured" | Add `[microsoft] client_id` to config.toml |
| "Token has stale IMAP scope" | Re-authorize: `msgvault add-o365 you@domain.com` |
| UPN differs from SMTP address | msgvault logs a warning; if sync fails, use the UPN shown in the warning as the email argument |
| Token refresh timed out | Check network connectivity to `login.microsoftonline.com` |
| Port 8089 already in use | Another process is using the callback port — stop it and retry |
| Authorization fails with tenant error | Verify your Azure AD app supports the account type (personal vs org) |

### Personal vs Organizational Accounts

msgvault automatically detects account type from the email domain. The following domains are treated as **personal** Microsoft accounts:

- `hotmail.com`, `outlook.com`, `live.com`, `msn.com`
- Regional variants: `hotmail.co.uk`, `hotmail.fr`, `outlook.de`, `outlook.jp`, etc.

All other domains are treated as **organizational** (Microsoft 365 / Entra ID).

If your organization uses a custom domain that should be treated as personal (rare), or vice versa, the `tid` claim in the ID token is authoritative — msgvault corrects the IMAP scope automatically.

---

## Generic IMAP

Use IMAP to archive email from any standard IMAP server: Yahoo Mail, iCloud Mail, custom mail servers, or providers that don't have native OAuth support.

### Adding an Account

```bash
# Default: implicit TLS (IMAPS, port 993)
msgvault add-imap --host imap.example.com --username user@example.com

# STARTTLS on port 143
msgvault add-imap --host mail.example.com --username user@example.com --starttls

# Custom port
msgvault add-imap --host mail.example.com --port 993 --username user@example.com

# Plain text (not recommended — only for local testing)
msgvault add-imap --host localhost --port 143 --username user@example.com --no-tls
```

You will be prompted to enter your password interactively. For scripting, pipe the password via stdin:

```bash
read -s PASS && echo "$PASS" | msgvault add-imap --host imap.example.com --username user@example.com
```

### Connection Options

| Flag | Description | Default |
|------|-------------|---------|
| `--host` | IMAP server hostname (required) | — |
| `--port` | IMAP server port | 993 (TLS), 143 (STARTTLS) |
| `--username` | IMAP username / email (required) | — |
| `--starttls` | Use STARTTLS upgrade | — |
| `--no-tls` | Disable TLS (plain text) | — |

### Syncing

```bash
# Full sync
msgvault sync-full user@example.com

# Date range (uses IMAP SEARCH)
msgvault sync-full user@example.com --after 2024-01-01 --before 2024-12-31

# Note: IMAP does not support --query flag or incremental sync
```

### Security

- Passwords are stored on disk with `0600` permissions (owner-only read/write)
- Credentials are stored in `~/.msgvault/tokens/imap_<hash-prefix>.json`
- Use **app-specific passwords** instead of your primary account password when your provider supports them

### Provider-Specific IMAP Settings

| Provider | IMAP Host | Port | Auth | Notes |
|----------|-----------|------|------|-------|
| Gmail IMAP | `imap.gmail.com` | 993 | App password | Enable IMAP in Gmail settings; use app-specific password |
| Yahoo Mail | `imap.mail.yahoo.com` | 993 | App password | Generate app password at https://login.yahoo.com/account/security |
| iCloud Mail | `imap.mail.me.com` | 993 | App password | Generate at appleid.apple.com > Sign-In and Security |
| Outlook.com | `outlook.office365.com` | 993 | App password | Prefer `add-o365` for OAuth; IMAP fallback requires app password |
| Proton Mail | `imap.protonmail.ch` | 993 | App password | Requires Proton Mail Bridge for IMAP access |
| Fastmail | `imap.fastmail.com` | 993 | App password | Generate at Settings > Passwords & Security |
| Zoho Mail | `imap.zoho.com` | 993 | App password | Generate at Settings > Security > App Passwords |

### Troubleshooting

| Issue | Solution |
|-------|----------|
| Connection test failed | Verify hostname, port, and credentials; check firewall rules |
| "cannot read password" | Pipe password via stdin or ensure you have a terminal |
| No incremental sync | IMAP has no equivalent to Gmail's History API — full sync each time |
| `--query` not supported | IMAP only supports date filtering via `--after`/`--before` |
| Gmail IMAP authentication fails | Enable IMAP in Gmail settings; use an app-specific password, not your Google password |

---

## Multi-Account Setup

You can mix and match providers in a single msgvault database:

```bash
# Gmail account
msgvault add-account you@gmail.com
msgvault sync-full you@gmail.com

# Microsoft 365 account
msgvault add-o365 you@company.com
msgvault sync-full you@company.com

# Generic IMAP account
msgvault add-imap --host imap.oldprovider.com --username you@oldprovider.com
msgvault sync-full you@oldprovider.com
```

List all configured accounts:

```bash
msgvault list-accounts
```

### Scheduled Syncs

Configure automatic syncs in `config.toml`:

```toml
[[accounts]]
email = "you@gmail.com"
schedule = "0 2 * * *"   # Daily at 2am
enabled = true

[[accounts]]
email = "you@company.com"
schedule = "0 */4 * * *"  # Every 4 hours
enabled = true
```

Run the daemon to execute scheduled syncs:

```bash
msgvault serve
```

---

## Configuration Reference

### Full `config.toml` Example

```toml
# Data storage location
[data]
data_dir = "~/.msgvault"           # Override default data directory

# Gmail OAuth
[oauth]
client_secrets = "/path/to/client_secret.json"

# Named OAuth apps for Google Workspace
[oauth.apps.acme]
client_secrets = "/path/to/acme_secret.json"

# Microsoft 365 / Outlook.com
[microsoft]
client_id = "your-azure-app-client-id"
tenant_id = "common"               # Optional; defaults to "common"

# Sync settings
[sync]
rate_limit_qps = 5                 # Gmail API rate limit

# Scheduled syncs
[[accounts]]
email = "you@gmail.com"
schedule = "0 2 * * *"
enabled = true
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `MSGVAULT_HOME` | Override the default data directory (`~/.msgvault`) |

### Directory Structure

```
~/.msgvault/
├── config.toml              # Configuration file
├── msgvault.db              # SQLite database
├── tokens/                  # OAuth tokens and IMAP credentials
│   ├── you@gmail.com.json           # Gmail OAuth token
│   ├── microsoft_you@company.com.json  # Microsoft OAuth token
│   └── imap_<hash>.json             # IMAP password credential
├── attachments/             # Content-addressed attachment storage
└── analytics/               # Parquet cache for fast analytics
```

---

## Security Considerations

1. **Token files** are stored with `0600` permissions (owner-only)
2. **OAuth tokens** are auto-refreshed and persisted atomically (temp file + rename)
3. **IMAP passwords** are stored as individual credential files with restricted permissions
4. **Microsoft tokens** are revoked on account removal (best-effort)
5. **Gmail tokens** are validated against the Gmail Profile API to prevent account mismatch
6. **PKCE** is required for Microsoft OAuth flows (S256 code challenge)
7. **Nonce-based replay protection** is used for Microsoft ID tokens
8. **CSRF protection** via random state tokens in all OAuth flows

### Headless / Server Deployment

For servers without a browser:

- **Gmail:** Authorize on a desktop machine, copy the token file via `scp`
- **Microsoft 365:** Browser authorization is required; no headless/device flow is currently supported
- **IMAP:** Fully headless — pipe the password via stdin

---

## Migration Between Providers

If you're switching from one provider to another (e.g., Google Workspace to Microsoft 365), you can import your old email via MBOX export:

```bash
# Export from old provider as MBOX
# Then import into msgvault
msgvault import-mbox you@olddomain.com /path/to/export.mbox

# Set up new provider
msgvault add-o365 you@newdomain.com
msgvault sync-full you@newdomain.com
```

Both accounts coexist in the same database and can be queried together.
