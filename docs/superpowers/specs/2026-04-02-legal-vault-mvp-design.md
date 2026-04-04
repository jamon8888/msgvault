# Legal Vault - MVP Architecture Design

**Date:** 2026-04-02
**Status:** Draft
**Project:** Email Archiving Solution (SaaS + On-Premise)

---

## 1. Executive Summary

Ce document définit l'architecture technique du MVP pour **Legal Vault**, une solution d'archivage email conçue pour le marché européen avec deployment **SaaS** et **On-Premise**.

Le code source (Socle Commun) est rigoureusement identique entre les deux déploiements. Seuls la configuration et l'infrastructure diffèrent.

### Objectifs MVP
- Ingestion SMTP (Journaling)
- Chiffrement/Crypto-shredding (RGPD)
- Recherche eDiscovery
- Export pour audits

---

## 2. Architecture Globale

```
┌─────────────────────────────────────────────────────────────────┐
│                        Legal Vault                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────┐  │
│  │ SMTP Server  │───▶│ Crypto Layer │───▶│ Storage Adapter  │  │
│  │ (emersion)   │    │  (minio/sio) │    │ (S3/MinIO)       │  │
│  └──────────────┘    └──────────────┘    └──────────────────┘  │
│                                                      │           │
│  ┌──────────────┐    ┌──────────────┐               │           │
│  │  SQLite/    │◀───│   Search     │◀──────────────┘           │
│  │  Parquet    │    │  (FTS5)      │                            │
│  └──────────────┘    └──────────────┘                            │
│                                                      │           │
│  ┌──────────────────────────────────────────────────▼─────────┐  │
│  │                    API HTTP Server                      │  │
│  │              (eDiscovery + Admin)                       │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. Composants du Socle Commun

### 3.1 SMTP Server (Ingestion)

**Librairie:** `github.com/emersion/go-smtp`

**Responsabilités:**
- Écouter sur port SMTP configurable (défaut: 2525)
- Accepter les emails en entrée (Journaling mode)
- Valider l'expéditeur (whitelist configurable)
- Passer les emails bruts au layer crypto

**Configuration:**
```go
type SMTPConfig struct {
    ListenAddr string // "0.0.0.0:2525"
    Hostname   string // "archive.company.com"
    
    // TLS Configuration
    TLSEnabled     bool   // Activer TLS
    TLSCertFile    string // Path vers cert.pem
    TLSKeyFile     string // Path vers key.pem
    ForceTLS       bool   // Rejeter si pas de TLS (pour compliance)
    
    // Authentification optionnelle (pour SaaS multi-tenant)
    EnableAuth bool
    AuthUsers  map[string]string // username -> password
    
    // Relay mode (pas d'auth, juste receive)
    RelayMode bool
}
```

**Interface interne:**
```go
type EmailProcessor interface {
    ProcessEmail(ctx context.Context, from string, rcpt []string, data []byte) error
}
```

---

### 3.2 Crypto-Shredding Layer

**Librairies:**
- Chiffrement: `github.com/minio/sio` (DARE format)
- Hachage: Standard library `crypto/sha256`
- Génération clé: `crypto/rand`

**Concept Crypto-shredding (Corrigé):**
1. **Génération clé aléatoire** - Clé AES-256 générée aléatoirement (UUID)
2. **Chiffrement contenu** - Chiffrer l'email avec AES-256-GCM
3. **Hachage** - Générer SHA-256 du contenu original (pour indexing)
4. **Stockage clé chiffrée** - Chiffrer la clé avec master key (HSM ou fichier)
5. **Stockage données** - Stocker: hash ID + données chiffrées + clé chiffrée

**Implementation:**
```go
type CryptoShredder interface {
    // Shred chiffre et hache un email
    Shred(ctx context.Context, emailRaw []byte, tenantID string) (*ShreddedEmail, error)
    
    // Unshred déchiffre (pour recovery/admin)
    Unshred(ctx context.Context, id string, tenantID string) ([]byte, error)
    
    // Delete supprime définitivement (droit à l'oubli)
    Delete(ctx context.Context, id string, tenantID string) error
}

type ShreddedEmail struct {
    ID             string    // SHA-256 hash (content ID pour recherche)
    EncryptedData  []byte   // Données chiffrées (minio/sio)
    EncryptedKey   []byte   // Clé chiffrée avec master key
    KeyID          string   // UUID de la clé (pour lookup)
    Metadata       EmailMeta // Métadonnées non chiffrées
}

// KeyInfo stocke les métadonnées de clé (pour lookup et audit)
type KeyInfo struct {
    KeyID        string    // UUID unique
    EncryptedKey []byte   // Clé AES chiffrée par master key
    CreatedAt    time.Time
    TenantID     string   // Pour isolation multi-tenant
}

// MasterKeyHandler gère les clés maître (HSM ou fichier)
type MasterKeyHandler interface {
    EncryptKey(ctx context.Context, plainKey []byte) ([]byte, error)
    DecryptKey(ctx context.Context, encryptedKey []byte) ([]byte, error)
    RotateKey(ctx context.Context) error  // Pour rotation
}

type EmailMeta struct {
    ReceivedAt   time.Time
    From         string
    To           []string
    Subject      string // Optionnel: peut être aussi chiffré
    MessageID    string
    TenantID     string // Pour multi-tenant SaaS
}
```

**Key Escrow (Critical pour conformité):**
```go
// Pour On-Premise: fichier de backup chiffré avec recovery key
// Pour SaaS: HSM (Thales Luna, AWS CloudHSM, Azure Key Vault)
type KeyEscrow struct {
    // Backup de la master key chiffrée avec recovery key séparée
    BackupEncryptedKey []byte
    RecoveryKeyID      string
    CreatedAt          time.Time
    BackupLocation     string  // "s3://backup/keys" ou "/secure/backup"
}
```

**Droit à l'oubli (RGPD):**
```go
// Delete = supprimer données + clé = irréversible
// Note: le hash (ID) reste pour historique d'audit mais données sont perdues
func (c *CryptoShredder) Delete(ctx context.Context, id, tenantID string) error {
    // 1. Supprimer les données chiffrées du storage
    // 2. Supprimer la clé correspondante
    // 3. Logger l'événement pour audit trail
    // 4. Retourner confirmation (avec hash pour référence)
}
```

**Conformité RGPD:**
- ✅ Le contenu original ne peut jamais être récupéré sans la clé
- ✅ Suppression complète = "droit à l'oubli" exécuté
- ✅ Clé de backup pour recovery (HSM recommandé en production)
- ✅ Audit trail de toutes les opérations

---

### 3.3 Storage Adapter (S3/MinIO)

**Librairie:** `github.com/minio/minio-go/v7`

**Responsabilités:**
- Abstraction du stockage objet (S3 cloud ou MinIO local)
- Support WORM (Write Once Read Many)
- Gestion du lifecycle (tiering froid)

**Interface:**
```go
type StorageAdapter interface {
    // Put stocke un objet
    Put(ctx context.Context, key string, data []byte, opts PutOptions) error
    
    // Get récupère un objet
    Get(ctx context.Context, key string) ([]byte, error)
    
    // Delete supprime (ou marque pour suppression)
    Delete(ctx context.Context, key string) error
    
    // Exists vérifie l'existence
    Exists(ctx context.Context, key string) (bool, error)
    
    // List liste les objets
    List(ctx context.Context, prefix string) ([]string, error)
}

type PutOptions struct {
    ContentType  string
    Metadata     map[string]string
    WORM         bool  // Enable WORM lock
}
```

**Configuration:**
```go
type StorageConfig struct {
    // Type de backend
    BackendType string // "s3" ou "minio"
    
    // S3 Cloud (pour SaaS)
    S3Endpoint        string
    S3Region          string
    S3AccessKey       string
    S3SecretKey       string
    S3Bucket          string
    S3ObjectLock      bool  // Pour WORM cloud
    
    // MinIO local (pour On-Premise)
    MinIOEndpoint     string
    MinIOAccessKey    string
    MinIOSecretKey    string
    MinIOBucket       string
    MinIOWORM          bool  // Enable MinIO WORM
    MinIODrivePath    string // Path vers data directory
}
```

---

### 3.4 Search Engine (Existant msgvault)

Le moteur de recherche msgvault est déjà complet:
- **SQLite + FTS5** - Recherche full-text
- **DuckDB/Parquet** - Analytics rapide

**Intégration:**
Le layer SMTP stocke les métadonnées dans SQLite via l'API existante de msgvault. Le contenu chiffré est stocké dans S3/MinIO.

```go
// Nouvel endpoint pour intégrer les emails reçus par SMTP
type IngestionService struct {
    store     *store.Store
    crypto    CryptoShredder
    storage   StorageAdapter
}
```

---

## 4. Déploiement SaaS vs On-Premise

### 4.1 Configuration Commune

```yaml
# config.yaml - Identique pour les deux déploiements
app:
  name: legal-vault
  version: 1.0.0
  
smtp:
  listen: "0.0.0.0:2525"
  hostname: "archive.example.com"
  relay_mode: true
  
crypto:
  algorithm: "AES-256-GCM"
  key_path: "/etc/legal-vault/keys"
  
search:
  fts_enabled: true
  parquet_enabled: true
```

### 4.2 Configuration SaaS (Cloud)

```yaml
# config.saaas.yaml
deployment:
  mode: "saas"
  
storage:
  backend: "s3"
  endpoint: "s3.eu-west-1.amazonaws.com"
  region: "eu-west-1"
  bucket: "legal-vault-archives"
  object_lock: true  # WORM via S3
  
multi_tenant:
  enabled: true
  isolation: "namespace"  # K8s namespace per tenant
  
scaling:
  replicas: 3
  autoscaling: true
```

### 4.3 Configuration On-Premise

```yaml
# config.onprem.yaml
deployment:
  mode: "onpremise"
  
storage:
  backend: "minio"
  endpoint: "localhost:9000"
  bucket: "archives"
  worm: true
  data_path: "/data/archives"
  
scaling:
  replicas: 1
  autoscaling: false
```

---

## 5. Format de Distribution

### 5.1 On-Premise (Appliance)

**Option A: Docker Compose**
```yaml
# docker-compose.yml
services:
  legal-vault:
    image: legal-vault/agent:latest
    ports:
      - "2525:2525"  # SMTP
      - "8080:8080"  # HTTP API
    volumes:
      - ./config:/etc/legal-vault
      - ./data:/data
      - ./keys:/keys  # Clés crypto
    environment:
      - CONFIG_FILE=/etc/legal-vault/config.yaml
      
  minio:
    image: minio/minio:latest
    command: server /data --worm
    volumes:
      - ./minio-data:/data
```

**Option B: Binary unique**
- Binaire Go unique + fichier config
- MinIO compilé en embedded (ou exécutable séparé)

---

## 6. Flux de Données

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Serveur   │     │   Crypto    │     │   Storage   │     │   SQLite    │
│    SMTP     │────▶│  Shredding  │────▶│   (S3/MinIO)│     │  (Index)    │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
       │                   │                   │                   │
       │ Raw Email         │ Encrypted         │ Key (if stored)   │ Metadata
       │                   │ + Hash            │                   │
       ▼                   ▼                   ▼                   ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Validation  │     │ AES-256-GCM │     │  Bucket     │     │  FTS5       │
│ + Queue     │     │ + SHA-256   │     │  WORM       │     │  Index      │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
```

---

## 7. API HTTP (eDiscovery)

Étendre l'API existante msgvault:

```go
// Nouveaux endpoints

// POST /api/v1/emails/search
type SearchRequest struct {
    Query       string   `json:"query"`
    TenantID    string   `json:"tenant_id"`  // SaaS only
    DateFrom    string   `json:"date_from"`  // YYYY-MM-DD
    DateTo      string   `json:"date_to"`
    Limit       int      `json:"limit"`
    Offset      int      `json:"offset"`
}

// GET /api/v1/emails/:id/export
type ExportRequest struct {
    Format      string   `json:"format"`  // "eml", "pdf"
    IncludeAttachments bool   `json:"include_attachments"`
}
```

---

## 8. Sécurité

| Aspect | Implémentation |
|--------|----------------|
| **Chiffrement au repos** | AES-256-GCM via minio/sio |
| **Transport** | TLS 1.3 (SMTP + HTTP) |
| **Clés** | Fichier séparé ou HSM |
| **WORM** | S3 Object Lock ou MinIO WORM |
| **Audit** | Logs immuables, syscall |
| **Access Control** | RBAC via API Key |

---

## 9. Plan d'Implémentation

### Phase 1: SMTP Ingestion (Semaine 1-2)
1. Implémenter serveur SMTP basic
2. Intégrer avec msgvault store
3. Tests d'intégration

### Phase 2: Crypto-Shredding (Semaine 3)
1. Layer chiffrement avec minio/sio
2. Génération et gestion des clés
3. Opérations shred/unshred

### Phase 3: Storage Adapter (Semaine 4)
1. Interface StorageAdapter
2. Implémentation S3
3. Implémentation MinIO

### Phase 4: Configuration (Semaine 5)
1. Modes SaaS/On-Premise
2. Docker Compose
3. OVA (optionnel)

### Phase 5: API eDiscovery (Semaine 6)
1. Endpoints recherche
2. Export emails
3. Tests E2E

---

## 10.2 Attachments Handling

```go
type AttachmentMeta struct {
    ID          string
    EmailID     string    // Référence vers l'email parent
    Filename    string
    ContentType string
    Size        int64
    Hash        string    // SHA-256 pour déduplication
}

// Les attachments sont:
// 1. Hachés pour déduplication (same content = same hash)
// 2. Chiffrés avec la même clé que l'email parent
// 3. Stockés dans le même bucket S3/MinIO
```

---

## 10.3 Multi-Tenant Isolation (SaaS)

```go
type TenantIsolation struct {
    // Isolation au niveau base de données
    // - Chaque tenant a son propre schéma SQLite
    // - Ou row-level security avec tenant_id
    
    // Isolation au niveau storage
    // - Bucket S3 par tenant (ex: legal-vault-{tenant-id})
    // - Ou prefix avec tenant_id (ex: {tenant-id}/emails/)
    
    // Isolation au niveau clé crypto
    // - Master key par tenant
    // - Ou clé maître globale + tenant ID dans le chiffré
}
```

---

## 10.4 Audit Trail

```go
type AuditEvent struct {
    EventID     string    // UUID
    Timestamp   time.Time
    TenantID    string    // Pour SaaS
    UserID      string    // Qui a fait l'action
    Action      string    // "email.stored", "email.viewed", "email.deleted", "key.rotated"
    ResourceID  string    // ID de la ressource concernée
    IPAddress   string
    Outcome     string    // "success", "failure"
    Details     string    // JSON avec détails additionnels
}

// Stockage: les logs d'audit sont:
// - Immuatbles (WORM)
// - Exportables vers SIEM externe
// - Conservés 10 ans (conformité légale)
```

---

## 10.5 Key Rotation

```go
type KeyRotation struct {
    Schedule    string    // "90d" = toutes les 90 jours
    Method      string    // "encrypt-all" ou "re-encrypt-on-access"
    
    // Pour "re-encrypt-on-access":
    // - L'ancienne clé déchiffre à la lecture
    // - Les données sont rechiffrées avec la nouvelle clé
    // - L'ancienne clé est supprimée après un grace period
}
```

---

## 11. Risques et Mitigations

| Risque | Mitigation |
|--------|-------------|
| Performance SMTP sous charge | Queue + workers async |
| Perte de clés crypto | HSM + backup sécurisé avec key escrow |
| Compliance RGPD | Audit trail, droit à l'oubli implémenté |
| Scalabilité recherche | Sharding SQLite si needed |
| Perte master key | Procédure de recovery avec encrypted backup |
| Clés perdues = données perdues | Key escrow avec recovery key séparée |

---

## 12. Conclusion

Cette architecture permet:
- ✅ **Code unique** pour SaaS et On-Premise
- ✅ **Conformité NIS2/RGPD** via crypto-shredding avec clé aléatoire
- ✅ **Key Escrow** pour recovery en cas de perte de clés
- ✅ **Droit à l'oubli** implémenté (suppression clé + données)
- ✅ **Flexibilité** deployment (Cloud, VM, Docker)
- ✅ **Audit Trail** pour traçabilité des opérations

---

## 13. Dépendances Go (Mis à jour)

```go
require (
    github.com/emersion/go-smtp v0.21.3
    github.com/minio/minio-go/v7 v7.0.74
    github.com/minio/sio v0.3.1
    github.com/spf13/cobra v1.9.1
    github.com/google/uuid v1.6.0  // Pour KeyID UUID
    github.com/wesm/msgvault v0.0.0  // Core existant
)
```

---

*Spec mis à jour le 2026-04-02 suite aux retours de review*
- ✅ **Scalabilité** via architecture modulaire

**Prochaine étape:** Approbation du design et passage à l'implémentation.
