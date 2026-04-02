# Guide de Configuration Legal Vault - Microsoft 365

**Version:** 1.0.0
**Date:** 2026-04-02
**cible:** Microsoft 365 / Exchange Online

---

## Table des Matières

1. [Architecture](#1-architecture)
2. [Prérequis](#2-prérequis)
3. [Configuration OAuth Azure AD](#3-configuration-oauth-azure-ad)
4. [Configuration du Journaling Exchange](#4-configuration-du-journaling-exchange)
5. [Configuration On-Premise](#5-configuration-on-premise)
6. [Configuration SaaS](#6-configuration-saas)
7. [Recherche eDiscovery](#7-recherche-ediscovery)
8. [Dépannage](#8-dépannage)

---

## 1. Architecture

### 1.1 Flux de Données

```
┌──────────────────────────────────────────────────────────────────────┐
│                        Microsoft 365                                  │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────┐     ┌──────────────┐     ┌────────────────────┐   │
│  │  Boîte aux  │────▶│   Journaling │────▶│  Legal Vault (SMTP)│   │
│  │   lettres   │     │   Exchange   │     │    :2525           │   │
│  └─────────────┘     └──────────────┘     └─────────┬──────────┘   │
│                                                      │               │
│                                                      ▼               │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                   Legal Vault                                │   │
│  │                                                              │   │
│  │   ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │   │
│  │   │ SMTP Server  │───▶│ Crypto Layer │───▶│  S3/MinIO    │  │   │
│  │   │  (Ingestion) │    │  (Chiffrement)│    │  (WORM)      │  │   │
│  │   └──────────────┘    └──────────────┘    └──────────────┘  │   │
│  │                                                      │        │   │
│  │   ┌──────────────┐    ┌──────────────┐               │        │   │
│  │   │   SQLite    │◀───│    FTS5      │◀──────────────┘        │   │
│  │   │  (Index)   │    │  (Recherche) │                        │   │
│  │   └──────────────┘    └──────────────┘                        │   │
│  │                                                              │   │
│  │   ┌───────────────────────────────────────────────────────▼───┐ │   │
│  │   │               API HTTP (eDiscovery)                      │ │   │
│  │   └───────────────────────────────────────────────────────────┘ │   │
│  └───────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────┘
```

### 1.2 Composants

| Composant | Rôle | Port Default |
|-----------|------|--------------|
| SMTP Server | Ingestion emails | 2525 |
| API HTTP | eDiscovery | 8080 |
| MinIO | Stockage WORM | 9000 |

---

## 2. Prérequis

### 2.1 Licence Microsoft 365 Requises

- **Exchange Online Plan 1** ou supérieur
- **Journaling** activé sur le tenant

### 2.2 Permissions Azure AD

Pour l'authentification OAuth2, vous avez besoin de:

1. Accès administrateur Azure AD
2. Enregistrement d'application (App Registration)
3. Permissions API pour IMAP.AccessAsUser.All

### 2.3 Configuration Réseau

| Service | Entrant | Sortant | Protocole |
|---------|---------|---------|-----------|
| SMTP | ✅ :2525 | - | TCP |
| API | ✅ :8080 | - | HTTP/HTTPS |
| S3/MinIO | - | ✅ :9000 | HTTPS |
| Azure AD | - | ✅ :443 | HTTPS |

---

## 3. Configuration OAuth Azure AD

### 3.1 Créer l'Application Azure

1. **Portail Azure** → **Azure Active Directory** → **App registrations**

2. **Nouvelle inscription:**
   - Nom: `Legal Vault Archive`
   - Comptes supportés: `Comptes dans un organisation uniquement`
   - Redirect URI: `http://localhost`

3. **API Permissions:**
   ```
   API Microsoft Graph:
   ├── IMAP.AccessAsUser.All (Delegated)
   ├── User.Read (Delegated)
   └── offline_access (Delegated)
   ```

4. **Activer Authentification OAuth2:**
   - Token ID: Cocher `ID tokens`
   - Tokens d'accès: Cocher `Access tokens`

5. **Générer Client Secret:**
   - Certificates & secrets → Nouveau secret client
   - Copier la valeur (elle disparaît après!)

### 3.2 Configuration Legal Vault

**config.yaml:**
```yaml
app:
  name: legal-vault
  version: 1.0.0

microsoft:
  client_id: "VOTRE_CLIENT_ID"
  client_secret: "VOTRE_CLIENT_SECRET"
  tenant_id: "VOTRE_TENANT_ID"  # Pour org-specific apps

smtp:
  listen: "0.0.0.0:2525"
  hostname: "archive.votre-domaine.com"
  relay_mode: true

storage:
  backend: "minio"
  endpoint: "localhost:9000"
  bucket: "archives"
  worm: true
  data_path: "/data/archives"

crypto:
  algorithm: "AES-256-GCM"
  key_path: "/etc/legal-vault/keys"

search:
  fts_enabled: true
```

### 3.3 Ajouter un Compte Microsoft 365

```bash
# Avec le binaire Legal Vault
./legal-vault add-o365 user@entreprise.com

# Pour compte organisationnel (O365)
./legal-vault add-o365 user@entreprise.com --tenant

# Output attendu:
# Opening browser for Microsoft authentication...
# Please authorize the application.
# Successfully added account: user@entreprise.com
```

---

## 4. Configuration du Journaling Exchange

### 4.1 Activer le Journaling Global

**Via Exchange Admin Center (EAC):**

1. **Compliance Management** → **Journaling**
2. **Nouveau** (+)
3. **Journaling recipient:** `archive@entreprise.com`
4. **Scope:** `External`
5. **Save**

### 4.2 Créer une Boîte aux Lettres de Journaling

```powershell
# PowerShell Exchange Online
New-Mailbox -Name "Legal Vault Archive" `
             -PrimarySmtpAddress archive@entreprise.com `
             -Archive `
             -RetentionPolicy "Default"
```

### 4.3 Configurer le Transport

**Via Exchange Admin Center:**

1. **Mail flow** → **Rules**
2. **Nouvelle règle:**
   - Nom: `Archive to Legal Vault`
   - Apply to: `Messages sent to`
   - Select recipients: *Tous les utilisateurs*
   - Do the following: `Redirect the message to` → `archive@entreprise.com`
   - Severity: `Low`

### 4.4 Configuration du Connecteur de Journaling

Pour rediriger vers le serveur Legal Vault SMTP:

```powershell
# Créer un connecteur de réception
New-ReceiveConnector -Name "Legal Vault In" `
                      -InternetBindings @{0=":2525"} `
                      -InternalEncoding UTF8
```

---

## 5. Configuration On-Premise

### 5.1 Déploiement Docker Compose

**Structure des fichiers:**
```
legal-vault/
├── config/
│   └── config.yaml
├── data/
├── keys/
├── docker-compose.yml
└── .env
```

**docker-compose.yml:**
```yaml
version: '3.8'

services:
  legal-vault:
    image: legal-vault/agent:latest
    ports:
      - "2525:2525"  # SMTP
      - "8080:8080"  # HTTP API
    volumes:
      - ./config:/etc/legal-vault
      - ./data:/data
      - ./keys:/keys
    environment:
      - CONFIG_FILE=/etc/legal-vault/config.yaml
      - LOG_LEVEL=info
    depends_on:
      - minio
    restart: unless-stopped
    networks:
      - legal-vault-net

  minio:
    image: minio/minio:latest
    command: server /data --worm --console-address ":9001"
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - ./minio-data:/data
    environment:
      - MINIO_ROOT_USER=${MINIO_USER}
      - MINIO_ROOT_PASSWORD=${MINIO_PASSWORD}
    restart: unless-stopped
    networks:
      - legal-vault-net

networks:
  legal-vault-net:
    driver: bridge
```

**.env:**
```bash
MINIO_USER=admin
MINIO_PASSWORD=votre-mot-de-passe-securise
```

### 5.2 Démarrage

```bash
# Démarrer les services
docker-compose up -d

# Vérifier le statut
docker-compose ps

# Logs
docker-compose logs -f legal-vault

# Vérifier que SMTP écoute
nc -zv localhost 2525
```

### 5.3 Configuration Réseau

**Pour accès depuis Internet:**

1. **Pare-feu:**
   - Entrant: TCP 2525 (SMTP)
   - Entrant: TCP 8080 (API optionnel)

2. **DNS:**
   - `archive.votre-domaine.com` → IP publique

3. **TLS (Recommandé):**
   ```yaml
   smtp:
     listen: "0.0.0.0:2525"
     hostname: "archive.votre-domaine.com"
     tls_enabled: true
     tls_cert_file: "/etc/legal-vault/certs/cert.pem"
     tls_key_file: "/etc/legal-vault/certs/key.pem"
     force_tls: true
   ```

---

## 6. Configuration SaaS

### 6.1 Architecture SaaS

```
┌─────────────────────────────────────────────────────────────────┐
│                      Cloud Provider EU                           │
│                   (OVH, Scaleway, 3DS OUTSCALE)                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    Kubernetes Cluster                      │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │  │
│  │  │   Ingress    │  │  Legal Vault │  │   MinIO     │    │  │
│  │  │ (NLB :2525) │  │    Pod       │  │   Pod       │    │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘    │  │
│  │                                                            │  │
│  │  ┌──────────────┐  ┌──────────────┐                      │  │
│  │  │    SQLite    │  │   DuckDB     │                      │  │
│  │  │  (Persistent)│  │  (Parquet)   │                      │  │
│  │  └──────────────┘  └──────────────┘                      │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    S3 Object Storage                       │  │
│  │              (avec Object Lock WORM)                      │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 6.2 Configuration Kubernetes

**deployment.yaml:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: legal-vault
  namespace: legal-vault
spec:
  replicas: 3
  selector:
    matchLabels:
      app: legal-vault
  template:
    metadata:
      labels:
        app: legal-vault
    spec:
      containers:
      - name: legal-vault
        image: legal-vault/agent:latest
        ports:
        - containerPort: 2525
        - containerPort: 8080
        env:
        - name: CONFIG_FILE
          value: /etc/legal-vault/config.yaml
        volumeMounts:
        - name: config
          mountPath: /etc/legal-vault
        - name: keys
          mountPath: /keys
      volumes:
      - name: config
        configMap:
          name: legal-vault-config
      - name: keys
        secret:
          secretName: legal-vault-keys
---
apiVersion: v1
kind: Service
metadata:
  name: legal-vault-smtp
  namespace: legal-vault
spec:
  type: LoadBalancer
  selector:
    app: legal-vault
  ports:
  - name: smtp
    port: 2525
    targetPort: 2525
    protocol: TCP
```

### 6.3 Configuration Multi-Tenant

```yaml
# config.saaas.yaml
deployment:
  mode: "saas"
  multi_tenant: true

storage:
  backend: "s3"
  endpoint: "s3.eu-west-1.amazonaws.com"
  region: "eu-west-1"
  bucket: "legal-vault-{tenant}"
  object_lock: true

tenants:
  - id: "tenant-1"
    name: "Entreprise A"
    domain: "entreprise-a.com"
  - id: "tenant-2"  
    name: "Entreprise B"
    domain: "entreprise-b.com"
```

### 6.4 Isolation des Tenants

| Niveau | Isolation | Méthode |
|--------|-----------|---------|
| Stockage | Bucket par tenant | `legal-vault-{tenant-id}` |
| Clés crypto | Clé maître par tenant | HSM partitionné |
| Base de données | Schema par tenant | `tenant_{id}.*` |
| Réseau | Namespace K8s | NetworkPolicy |

---

## 7. Recherche eDiscovery

### 7.1 API REST

**Base URL:** `http://localhost:8080/api/v1`

#### Rechercher des Emails

```bash
# Recherche simple
curl -X POST "http://localhost:8080/api/v1/emails/search" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "contrat",
    "date_from": "2024-01-01",
    "date_to": "2024-12-31",
    "limit": 50
  }'
```

**Réponse:**
```json
{
  "emails": [
    {
      "id": "sha256hash...",
      "subject": "Contrat de prestation",
      "from": "john@entreprise.com",
      "to": ["legal@entreprise.com"],
      "date": "2024-03-15T10:30:00Z",
      "snippet": "...contrat de prestation de services..."
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

#### Exporter un Email

```bash
curl -X GET "http://localhost:8080/api/v1/emails/sha256hash.../export?format=eml" \
  -o email-export.eml
```

### 7.2 Interface Web (TUI)

```bash
# Lancer l'interface TUI
./legal-vault tui

# Recherche interactive: /
# Filtrer par date: tapez 't' pour vue temporelle
# Filtrer par domaine: tapez 'a' pour account filter
```

### 7.3 Recherche Avancée

```bash
# Opérateurs de recherche
# AND (par défaut)
msgvault search "contrat AND facture"

# OR
msgvault search "contrat OR devis"

# Phrase exacte
msgvault search "\"contrat de location\""

# Champ spécifique
msgvault search "from:john@entreprise.com"
msgvault search "subject:facture"
```

---

## 8. Dépannage

### 8.1 Erreurs Courantes

| Erreur | Cause | Solution |
|--------|-------|----------|
| `550 5.7.1 Service not available` | SMTP non autorisé | Vérifier firewall, autoriser IP Exchange |
| `Authentication failed` | OAuth token expiré | Relancer `add-o365` |
| `WORM violation` | Tentative de suppression | WORM empêche toute suppression |
| `Connection refused :2525` | Service non démarré | `docker-compose restart legal-vault` |
| `TLS required but not offered` | ForceTLS activé | Configurer TLS ou désactiver |

### 8.2 Vérification des Services

```bash
# Vérifier SMTP
echo "QUIT" | nc -w 5 localhost 2525

# Vérifier API
curl -s http://localhost:8080/health

# Vérifier MinIO
mc alias set local http://localhost:9000 admin password
mc ls local/archives

# Logs Legal Vault
docker-compose logs -f --tail=100 legal-vault
```

### 8.3 Rotation des Clés

```bash
# Sauvegarder les clés actuelles
cp -r /keys /keys-backup-$(date +%Y%m%d)

# Générer nouvelle clé maître
./legal-vault crypto rotate-master-key

# Réencrypter tous les emails (processus long)
./legal-vault crypto reencrypt-all
```

### 8.4 Monitoring

```yaml
# Configuration Prometheus
metrics:
  enabled: true
  port: 9090
  
# Métriques disponibles:
# - legal_vault_emails_received_total
# - legal_vault_emails_stored_total  
# - legal_vault_storage_size_bytes
# - legal_vault_search_queries_total
```

---

## Annexe A: Variables d'Environnement

| Variable | Description | Défaut |
|----------|-------------|--------|
| `CONFIG_FILE` | Chemin config | `./config.yaml` |
| `LOG_LEVEL` | Niveau de log | `info` |
| `MSGVAULT_HOME` | Répertoire data | `~/.msgvault` |
| `SMTP_PORT` | Port SMTP | `2525` |
| `API_PORT` | Port API | `8080` |

---

## Annexe B: Commandes Utiles

```bash
# Ajouter un compte O365
./legal-vault add-o365 user@entreprise.com

# Sync manuel
./legal-vault sync-full user@entreprise.com --limit 100

# Démarrer le serveur
./legal-vault serve-archive

# Recherche CLI
./legal-vault search "facture"

# Stats
./legal-vault stats

# Rebuild index
./legal-vault build-cache --full-rebuild
```

---

## Annexe C: Checklist de Déploiement

- [ ] Azure App Registration créée
- [ ] Client secret généré et sécurisé
- [ ] Permissions API accordées
- [ ] Journaling Exchange configuré
- [ ] Boîte aux lettres de archive créée
- [ ] Connecteur réseau ouvert (TCP 2525)
- [ ] DNS configuré
- [ ] TLS configuré (production)
- [ ] MinIO démarré avec WORM
- [ ] Premier email de test reçu
- [ ] Recherche fonctionnelle

---

*Document généré pour Legal Vault v1.0.0*
