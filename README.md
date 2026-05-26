# mitm-gui

> **Version 0.1.0** - Windows desktop application for encrypted configuration loading, multi-format data ingestion, JSON Schema validation, and authenticated HTTP upload.

---

## Overview

`mitm-gui` is a Go + [Fyne](https://fyne.io/) desktop application that processes HR, ORG, and GEO master data through a four-step workflow:

1. **Load** data from CSV, Excel, or Kafka
2. **Validate** the data against a JSON Schema
3. **Upload** the data via HTTP POST (with optional OAuth2-like SaaS authentication)
4. **Audit** every step to a per-topic log file

The application reads its configuration from an **AES-256-GCM encrypted INI file** that can be freely distributed - the decryption key is user-supplied at runtime.

---

## Features

### Encrypted Configuration
- AES-256-GCM encryption with Argon2id key derivation (time=3, memory=64MB)
- Encrypted INI file contains all connection parameters (DB, Kafka, schema paths, upload URLs)
- Separate per-user encrypted proxy configuration
- Magic-byte file header (`MITM`) for quick format detection

### Multi-Format Data Loading

| Format | Library | Status |
|---|---|---|
| CSV | Go encoding/csv | Done |
| Excel (.xlsx) | excelize/v2 | Done |
| Kafka | segmentio/kafka-go | Done (SASL/SSL, TLS) |

### JSON Schema Validation
- Uses santhosh-tekuri/jsonschema/v5 (Draft 2020-12)
- Validates each record individually; collects all errors per row

### Upload with SaaS Auth
- **Direct POST** - simple HTTP upload with optional proxy
- **Authenticated upload** - two-legged OAuth2-like flow:
  1. POST /api/refreshtoken -> refresh token
  2. GET /api/token/ -> access token (Bearer)
  3. POST /api/employeeimport -> upload with Bearer token
- Automatic retry on 401 (token refresh)
- Proxy support for all HTTP calls (configurable per topic)

### Audit Logging
- Per-topic audit logs (e.g. 2026-05-26_hr_audit.log)
- All workflow steps are recorded: DATA_LOAD, VALIDATE, UPLOAD_START, UPLOAD_RESULT
- Dual logging: file + GUI summary panel

### Workflow State Machine
- Formal state machine (Idle -> ConfigLoaded -> DataLoaded -> ValidatedOk -> UploadedOk)
- Enforces valid transitions (no Validate before Load, no Upload before successful Validate)
- State-change callbacks update GUI buttons automatically

---

## Installation

### Prerequisites

- **Go 1.23+** (tested with 1.26.3)
- **C compiler** (for Fyne GUI on Windows):
  - TDM-GCC or scoop install gcc (recommended)
- **Windows** (primary target; cross-platform possible but untested)

### Build

```powershell
git clone https://github.com/zheng-bote/go_mitm-gui.git
cd go_mitm-gui

# Build both binaries
build.bat build

# Or manually
set CGO_ENABLED=1
go build -o mitm-gui.exe ./cmd/mitm-gui
go build -o encryptini.exe ./cmd/encryptini
```

### Run

```powershell
# GUI mode (default)
.mitm-gui.exe

# CLI mode (dump config for testing)
.mitm-gui.exe -cli -config=data/mitm-gui_prd.enc -password=prd-pass
```

---

## Configuration

### Encrypted INI File

The .enc file is an INI configuration encrypted with AES-256-GCM.

**Structure:**

```ini
[GLOBAL]
Admins = "user1", "user2"
log_level = "INFO"
log_file_path = "C:\logs\<yyyy-mm-dd>_app.log"

[HR]
INPUT_CSV = true
INPUT_EXCEL = true
JSON_SCHEMA_PATH = "C:\schemas\hr_schema_v1.json"
UPLOAD_URL = "http://localhost:8000/api/hr/upload"
audit_log_enabled = true
audit_log_file_path = "C:\audit_logs\<yyyy-mm-dd>_hr_audit.log"

[HR-INPUT-CSV]
DEFAULT_FILE_PATH = "C:\input\hr_data.csv"

[HR-AUTH]
saas_base_url = "https://bmwgroup.demo.cority.com"
saas_login = "Cority_Integration"
saas_password = "Cority123$"

[HR-UPLOAD]
use_proxy = true
proxy_host = "proxy.muc:8080"
upload_url = "https://bmwgroup.demo.cority.com/api/employeeimport"
upload_timeout = 120
```

**Encrypt an INI file:**

```powershell
encryptini.exe -password=<master-password> -in=data/config.ini -out=data/config.enc
```

### Proxy Configuration

Proxy settings are stored encrypted in a per-user file:

```
proxy_<OS-USERNAME>.enc
```

Saved next to the selected INI file, encrypted with its own password. Settings can be entered and saved from the GUI Settings tab.

---

## Usage

### Workflow

1. **Settings tab**: Select encrypted INI file -> enter master password -> Load Configuration
2. **Workflow tab** (auto-activated):
   - Select Topic (HR / ORG / GEO)
   - Select Input Format (CSV / Excel / Kafka)
   - For CSV/Excel: select file via dialog
   - Click Load Data -> records are parsed and displayed in the summary
   - Click Validate -> data is validated against the topic's JSON Schema
   - Click Upload -> data is uploaded (directly or with auth token)

### Build Commands

```powershell
build.bat tidy      # go mod tidy
build.bat build     # build all binaries
build.bat test      # run all tests
build.bat all       # tidy -> build -> test
build.bat cli       # CLI config dump
build.bat clean     # remove binaries and cache
```

---

## Project Structure

```
cmd/
  encryptini/       INI encryption CLI tool
  mitm-gui/         Application entry point
data/               Sample configs, schemas, encrypted files
gui/
  app.go            Fyne app setup, window layout
  settings.go       Settings tab (INI selection, proxy config)
  workflow.go       Workflow tab (load, validate, upload)
internal/
  config/           Encrypted INI loader
  crypto/           AES-256-GCM + Argon2id
  input/            DataSource interface (CSV, Excel, Kafka)
  logging/          Dual logger (file + GUI callback)
  model/            Data structures
  schema/           JSON Schema validation
  upload/           HTTP client + auth flow
  workflow/         State machine
schemas/            JSON Schema files (*.json)
```

---

## Testing

```powershell
go test ./...
```

Tests cover:
- Crypto round-trip (encrypt -> decrypt)
- INI parsing (including AUTH and UPLOAD sections)
- CSV data loading
- JSON Schema validation (valid, invalid, empty)
- HTTP upload (success, error, proxy)
- Auth flow (refresh -> access -> upload)
- Workflow state transitions

---

## Dependencies

| Package | Purpose |
|---|---|
| fyne.io/fyne/v2 | Desktop GUI |
| github.com/xuri/excelize/v2 | Excel (.xlsx) reader |
| github.com/santhosh-tekuri/jsonschema/v5 | JSON Schema validator |
| github.com/segmentio/kafka-go | Kafka consumer |
| golang.org/x/crypto | Argon2id key derivation |
| gopkg.in/ini.v1 | INI file parser |

---

## Author

ZHENG Robert - Initial development and architecture.
