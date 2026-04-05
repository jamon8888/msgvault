# Legal Vault

Legal Vault is msgvault's email ingestion and archival mode for organizations that need to journal, encrypt, and store email with RGPD (GDPR) compliance.

## Overview

Legal Vault provides:

- **SMTP ingestion server**: Receives journaling copies of email from your mail server
- **Crypto-shredding**: AES-256-GCM encryption with per-message keys
- **Content-addressed storage**: Deduplicated by SHA-256 hash
- **WORM support**: Immutable storage via MinIO for compliance
- **RGPD compliance**: Delete encryption keys to make data unrecoverable (right to be forgotten)

## Architecture

```
Exchange/Postfix ──SMTP──► msgvault serve-archive ──► Encrypted Storage (MinIO/S3)
                              │
                              ├── Crypto-shredding (AES-256-GCM)
                              ├── Per-message encryption keys
                              └── Content-addressed deduplication
```

## Quick Start

### Basic SMTP Server

```bash
msgvault serve-archive --smtp-host mail.example.com
```

This starts an SMTP server on port 2525 that accepts journaling copies of email, encrypts them, and stores them.

### With MinIO Storage

```bash
msgvault serve-archive \
  --smtp-host mail.example.com \
  --storage minio \
  --minio-endpoint localhost:9000 \
  --minio-bucket archives
```

### With WORM (Immutable Storage)

```bash
msgvault serve-archive \
  --smtp-host mail.example.com \
  --storage minio \
  --worm
```

WORM (Write Once, Read Many) ensures stored emails cannot be modified or deleted, meeting compliance requirements for immutable audit trails.

## CLI Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--smtp-host` | string | (required) | SMTP server hostname |
| `--smtp-port` | int | 2525 | SMTP listen port |
| `--storage` | string | `minio` | Storage backend: `s3` or `minio` |
| `--worm` | bool | false | Enable WORM (immutable storage) |
| `--minio-endpoint` | string | `localhost:9000` | MinIO endpoint |
| `--minio-bucket` | string | `archives` | MinIO bucket name |
| `--minio-data-path` | string | `/data/archives` | MinIO data path (on-premise) |
| `--s3-endpoint` | string | (required for S3) | S3 endpoint (cloud storage) |
| `--s3-bucket` | string | (required for S3) | S3 bucket name |

## Storage Backends

### MinIO (On-Premise)

MinIO is an S3-compatible object storage server that can run on-premise. Ideal for organizations that need full control over their data.

```bash
# Start MinIO (Docker)
docker run -d \
  -p 9000:9000 -p 9001:9001 \
  -v /data/minio:/data \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /data --console-address ":9001"

# Start Legal Vault
msgvault serve-archive \
  --smtp-host mail.example.com \
  --storage minio \
  --minio-endpoint localhost:9000 \
  --minio-bucket archives
```

### S3 (Cloud)

For cloud storage, use any S3-compatible service (AWS S3, Cloudflare R2, Backblaze B2, etc.).

```bash
msgvault serve-archive \
  --smtp-host mail.example.com \
  --storage s3 \
  --s3-endpoint s3.amazonaws.com \
  --s3-bucket my-email-archives
```

## Crypto-Shredding

Legal Vault encrypts each email with AES-256-GCM using a unique per-message key. This enables **crypto-shredding**: deleting the encryption key makes the data permanently unrecoverable, satisfying RGPD's right to be forgotten.

### How It Works

1. **Encryption**: Each email is encrypted with a random 256-bit AES-GCM key
2. **Key storage**: Keys are stored separately from encrypted data in `~/.msgvault/keys/`
3. **Content addressing**: Encrypted data is identified by SHA-256 hash of the original content
4. **Shredding**: Deleting a key (`crypto.Shredder.Delete()`) makes the corresponding data unrecoverable

### Current Limitations

- **Master key wrapping**: `FileKeyHandler.EncryptKey`/`DecryptKey` are currently pass-through — keys are stored unencrypted on disk. Master key encryption is planned but not yet implemented.
- **Unshred**: Decryption (`crypto.Shredder.Unshred()`) is not yet implemented.
- **Scope**: Crypto-shredding is only used by `serve-archive`. The main `sync-full` flow stores emails unencrypted in SQLite.

## SMTP Server Configuration

### TLS

The SMTP server supports TLS for secure email ingestion:

```bash
# TLS is configured programmatically in the SMTP server
# Certificates and keys are passed via the server config
```

### Authentication

Optional SMTP authentication can be enabled:

```go
smtp.Config{
    EnableAuth: true,
    AuthUsers: map[string]string{
        "journal": "secure-password",
    },
}
```

### Relay Mode

In relay mode, the SMTP server forwards journaling copies to the Legal Vault:

```
Exchange Journaling Rule ──SMTP──► Legal Vault SMTP Server
```

## Exchange Journaling Setup

To configure Microsoft Exchange to journal to Legal Vault:

1. Create a journaling rule in Exchange Admin Center
2. Set the journaling recipient to the Legal Vault SMTP address
3. Configure the SMTP server to accept connections from Exchange
4. Test with a sample email

## Postfix Integration

To configure Postfix to send journaling copies:

```
# /etc/postfix/main.cf
always_bcc = journal@legal-vault.example.com

# /etc/postfix/transport
legal-vault.example.com    smtp:[10.0.0.50]:2525
```

## eDiscovery API

The Legal Vault eDiscovery API for searching encrypted archives is planned but not yet implemented.

## Security Considerations

- SMTP server should be bound to internal networks only
- Use TLS for SMTP connections when possible
- Enable SMTP authentication for untrusted networks
- Keys are stored on disk with 0600 permissions
- Master key encryption is not yet implemented
- Encrypted data is content-addressed (SHA-256) for deduplication
