# Security Policy

## Reporting Vulnerabilities

If you discover a security vulnerability in msgvault, please report it responsibly:

1. **Do NOT open a public GitHub issue**
2. Email the maintainer directly or use GitHub's private vulnerability reporting feature
3. Include steps to reproduce, impact assessment, and any suggested fixes
4. Allow reasonable time for a fix before public disclosure

## Threat Model

### What msgvault protects

| Asset | Storage | Risk if compromised |
|-------|---------|-------------------|
| OAuth2 tokens | `~/.msgvault/tokens/` (per-account files) | Full Gmail API access to victim's account |
| Email bodies | SQLite database (`~/.msgvault/msgvault.db`) | Exposure of 20+ years of personal email |
| Attachments | Content-addressed files (`~/.msgvault/attachments/`) | Exposure of personal documents |
| Contact metadata | SQLite (participants table) | Social graph exposure |
| Search indexes | FTS5 virtual table in SQLite | Keyword-level exposure of email content |
| Analytics cache | Parquet files (`~/.msgvault/analytics/`) | Aggregate email metadata exposure |

### Security controls in place

**File permissions:**
- OAuth token files created with 0600 permissions (owner read/write only)
- Config directory (`~/.msgvault/`) should be 0700
- Attachment storage directory (`~/.msgvault/attachments/`) is created with 0700; attachment files are 0600
- Cross-platform support including Windows DACL

**SQL injection prevention:**
- All SQLite queries use parameterized statements via `database/sql`
- DuckDB queries over Parquet files use parameterized queries
- No string concatenation for query building

**Command injection prevention:**
- OAuth browser launch uses validated, well-formed URLs only
- No user-controlled input passed to `exec.Command` or shell execution

**Path traversal prevention:**
- Attachment storage uses content-hash addressing (SHA-256)
- Config paths resolved relative to a fixed base directory
- No user-controlled path components in file operations

**Input validation:**
- MIME parsing with charset detection (gogs/chardet) and safe encoding conversion
- Email addresses validated before database insertion
- Gmail API message IDs validated as alphanumeric

**PII filtering:**
- All MCP responses pass through a 3-pass PII detection pipeline before returning to the client.
- Pass 1: Structured PII (email, phone, IBAN, credit card, SSN, NIR) via wuming.
- Pass 2: Named entity recognition (PERSON, ORG, GPE, MONEY, DATE, etc.) via prose.
- Pass 3: Jurisdiction-specific legal patterns (FR, UK, US, DE) via regex.
- PII is replaced with descriptive tags (e.g., `[EMAIL]`, `[PHONE]`, `[PERSON]`) to preserve context.
- See [docs/pii-filtering.md](docs/pii-filtering.md) for full details.

**Crypto-shredding:**
- AES-256-GCM encryption with per-message keys for RGPD (GDPR) right-to-be-erasure compliance.
- Keys are stored separately from encrypted data; deleting a key renders data unrecoverable.
- Currently used by `serve-archive` (Legal Vault SMTP ingestion).
- Master key wrapping (EncryptKey/DecryptKey) is not yet implemented — keys are stored unencrypted on disk.
- Decryption (unshred) is not yet implemented.

### Known limitations

**Partial encryption at rest:**
- The SQLite database is not encrypted. Anyone with filesystem access to `~/.msgvault/` can read all archived emails.
- OAuth tokens are stored as plaintext JSON files (protected by file permissions only).
- **Crypto-shredding** is implemented for new email ingestion via `serve-archive` (AES-256-GCM with per-message keys), but unshred (decryption) is not yet implemented.
- Mitigation: Rely on OS-level full-disk encryption (FileVault, BitLocker, LUKS).

**PII filtering scope:**
- PII filtering applies only to MCP responses, not to the underlying database or TUI.
- The 3-pass pipeline (wuming + NER + legal regex) may miss edge cases or produce false positives.
- Legal pattern detection covers FR, UK, US, and DE jurisdictions only.

**CGO dependencies:**
- SQLite (mattn/go-sqlite3) and DuckDB (marcboeker/go-duckdb) use CGO, introducing native code that is harder to audit than pure Go.
- go-fitz (libmupdf) for PDF extraction requires CGO and is not available on Windows.
- Mitigation: Pin dependency versions, use govulncheck in CI, review updates via Dependabot.

**Gmail API deletion:**
- msgvault can stage and execute deletions via the Gmail API (trash or permanent delete).
- Mitigation: Deletion requires explicit user action, manifests are generated before execution, and the operation is logged.

## Automated Security Review

External pull requests are automatically reviewed by a Claude-powered security bot that checks for:
- Hardcoded secrets and credential exposure
- Command/SQL injection vulnerabilities
- Path traversal and file permission issues
- OAuth token handling problems
- Dependency supply chain risks (go.mod/go.sum changes)
- Workflow tampering (.github/ directory changes)

See [`.github/SECURITY_BOT.md`](.github/SECURITY_BOT.md) for details.

## Supply Chain

- **Dependabot** monitors Go modules and GitHub Actions for updates
- **govulncheck** runs on every PR (call-graph aware, Go vulnerability database)
- **CODEOWNERS** requires maintainer approval for go.mod, go.sum, and .github/ changes
- **GitHub Actions pinned to commit SHAs** to prevent tag-based supply chain attacks
