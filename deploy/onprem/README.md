# Legal Vault - On-Premise Deployment

Quick deployment using Docker Compose.

## Quick Start

```bash
# 1. Copy environment file
cp .env.example .env
# Edit .env with your values

# 2. Copy config
cp config.yaml config/config.yaml
# Edit config.yaml with your settings

# 3. Start services
docker-compose up -d

# 4. Check status
docker-compose ps
```

## Services

| Service | Port | Description |
|---------|------|--------------|
| Legal Vault SMTP | 2525 | Email ingestion |
| Legal Vault API | 8080 | eDiscovery API |
| MinIO Console | 9001 | Storage management |

## Configuration

### SMTP

Configure your email server to forward journaled emails:
- Host: `<your-server>:2525`
- Protocol: SMTP
- No authentication (relay mode)

### MinIO Console

Access at http://localhost:9001 to:
- Create buckets
- Manage retention policies
- View stored emails

## Data Directories

| Directory | Purpose |
|-----------|---------|
| `./data` | Email storage |
| `./keys` | Encryption keys |
| `./config` | Configuration |
| `./minio-data` | MinIO storage |

## Troubleshooting

```bash
# View logs
docker-compose logs -f legal-vault

# Restart services
docker-compose restart

# Stop services
docker-compose down
```
