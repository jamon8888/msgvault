# PII Filtering

msgvault automatically filters Personally Identifiable Information (PII) from all MCP server responses to prevent leakage of sensitive data when interacting with AI assistants like Claude Desktop.

## Overview

All MCP tool responses pass through a **3-pass PII detection pipeline** before being returned to the client. PII is replaced with descriptive tags (e.g., `[EMAIL]`, `[PHONE]`, `[PERSON]`) to preserve context while protecting sensitive data.

```
MCP Handler → Pass 1: Structured PII → Pass 2: NER → Pass 3: Legal Patterns → Client
```

## Pass 1: Structured PII (wuming)

Uses the [wuming](https://github.com/taoq-ai/wuming) library for zero-configuration detection of 75+ PII types across 14 locales.

| PII Type | Example | Replacement |
|----------|---------|-------------|
| Email addresses | `john@example.com` | `[EMAIL]` |
| Phone numbers | `+33 6 12 34 56 78` | `[PHONE]` |
| IBAN | `FR76 3000 6000 0112 3456 7890 189` | `[IBAN]` |
| Credit card numbers | `4111 1111 1111 1111` | `[CREDIT_CARD]` |
| SSN (US) | `123-45-6789` | `[SSN]` |
| NIR (French social security) | `1 85 12 75 108 123 45` | `[NIR]` |
| IP addresses | `192.168.1.1` | `[IP_ADDRESS]` |
| URLs | `https://example.com/page` | `[URL]` |

## Pass 2: Named Entity Recognition (prose)

Uses the [prose](https://github.com/tsawler/prose) library for named entity recognition. A pure Go implementation with no CGO dependencies.

| Entity Type | Example | Replacement |
|-------------|---------|-------------|
| PERSON | `John Smith` | `[PERSON]` |
| ORGANIZATION | `Acme Corporation` | `[ORGANIZATION]` |
| GPE (locations) | `Paris, France` | `[GPE]` |
| MONEY | `$1,234.56` | `[MONEY]` |
| DATE | `January 15, 2024` | `[DATE]` |
| TIME | `3:30 PM` | `[TIME]` |
| PERCENT | `15%` | `[PERCENT]` |
| FACILITY | `Eiffel Tower` | `[FACILITY]` |
| PRODUCT | `iPhone 15` | `[PRODUCT]` |
| EVENT | `World War II` | `[EVENT]` |
| WORK_OF_ART | `The Great Gatsby` | `[WORK_OF_ART]` |
| LANGUAGE | `English` | `[LANGUAGE]` |
| NORP | `American` | `[NORP]` |
| LAW | `GDPR` | `[LAW]` |
| ORDINAL | `first`, `2nd` | `[ORDINAL]` |
| CARDINAL | `42` | `[CARDINAL]` |

## Pass 3: Legal Pattern Detection

Regex-based detection of jurisdiction-specific legal identifiers. Supports four jurisdictions:

### French (fr)

- Case numbers: `RG n° 12/34567`, `n° 2023/12345`
- Judgment numbers: `jugement n° 2023-1234`
- Lawyer bar references: `CAPA`, `RIN` numbers
- CARPA account numbers
- Parquet numbers
- Settlement amounts
- RCS/SIRET company registrations
- Notary references
- Prisoner numbers
- Complaint/warrant numbers
- Land parcel references
- Mortgage references
- Insurance policy numbers
- Patent numbers (`EP 1 234 567`)
- Trademark references (INPI)
- Criminal record references
- Hospital stay references
- Medical record/certificate references

### UK

- Claim numbers: `HQ-2023-001234`
- Judgment citations: `[2023] EWHC 1234 (Ch)`
- SRA (Solicitors Regulation Authority) IDs
- Company numbers
- Land Registry title numbers
- Crime reference numbers
- NHS numbers
- National Insurance numbers

### US

- Federal case numbers: `1:23-cv-01234`
- SCOTUS numbers: `No. 22-1234`
- Docket numbers
- Slip opinions
- Bar numbers
- EINs (Employer Identification Numbers)
- Inmate numbers
- FBI numbers
- Medicare numbers
- MRNs (Medical Record Numbers)
- Patent numbers

### German (de)

- Aktenzeichen: `12 O 123/23`
- Staatsanwaltschaft references
- Urteil/Beschluss references
- RAK numbers
- Handelsregister: `HRB 12345`
- Grundbuch (land register) references
- Flurstück (parcel) numbers
- Notar references
- Gerichtsvollzieher (bailiff) IDs

## MCP Tools Affected

All MCP tools that return message data are filtered:

| Tool | Filtered Fields |
|------|----------------|
| `search_messages` | Subject, Snippet, FromEmail, FromName, Labels |
| `get_message` | BodyText, BodyHTML, all Address (Email, Name), Subject, Snippet, Labels |
| `list_messages` | Subject, Snippet, FromEmail, FromName |
| `get_attachment` | Attachment filename, chunk text |
| `search_attachments` | Attachment chunk text |
| `extract_attachment` | Extracted text content |
| `aggregate` | Sender emails, domain names |

## Configuration

PII filtering is enabled by default in the MCP server with the following settings:

```go
// internal/mcp/server.go
piiFilter, _ := pii.NewFilter(&pii.Config{
    LegalMode:   true,
    NERMode:     true,
    Jurisdictions: []string{"fr", "uk", "us", "de"},
})
```

### Custom Configuration

To customize PII filtering behavior, modify the MCP server initialization:

```go
// Disable legal pattern detection
pii.NewFilter(&pii.Config{
    LegalMode: false,
    NERMode:   true,
})

// Disable NER, keep only structured PII
pii.NewFilter(&pii.Config{
    LegalMode: false,
    NERMode:   false,
})

// Specific jurisdictions only
pii.NewFilter(&pii.Config{
    LegalMode:   true,
    NERMode:     true,
    Jurisdictions: []string{"us"}, // US only
})

// All jurisdictions (default)
pii.NewFilter(&pii.Config{
    LegalMode:   true,
    NERMode:     true,
    Jurisdictions: []string{"fr", "uk", "us", "de"},
})
```

### NER Entity Type Filtering

You can enable specific NER entity types:

```go
ner := pii.NewNERDetector("PERSON", "ORG", "GPE", "MONEY")
```

Empty types enables all entity types.

## Scope and Limitations

### What is filtered

- **MCP responses only**: All string fields returned by MCP tools
- **Message content**: Body text, HTML, subject, snippet
- **Addresses**: Sender and recipient email addresses and names
- **Labels**: Gmail labels
- **Attachment content**: Extracted text from PDFs, DOCX, TXT files
- **Account info**: Email addresses and display names

### What is NOT filtered

- **Database storage**: The underlying SQLite database is not modified
- **TUI output**: The interactive TUI shows unfiltered data
- **CLI output**: Command-line output is not filtered
- **API responses**: HTTP API responses from `serve` mode are not filtered
- **Request data**: Only responses are filtered, not incoming MCP requests

### Known limitations

- The 3-pass pipeline may miss edge cases or produce false positives
- Legal pattern detection covers only FR, UK, US, and DE jurisdictions
- NER accuracy depends on the prose library's capabilities
- Structured PII detection depends on wuming's pattern coverage
- Filtered data preserves structure but may lose some context

## Dependencies

| Library | Purpose | CGO Required |
|---------|---------|--------------|
| [wuming](https://github.com/taoq-ai/wuming) | Structured PII detection (75+ types, 14 locales) | No |
| [prose](https://github.com/tsawler/prose) | Named entity recognition | No |

Both libraries are pure Go with no CGO dependencies.
