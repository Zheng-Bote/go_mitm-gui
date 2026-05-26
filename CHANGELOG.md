# Changelog

## [0.1.0] - 2026-05-26

### Added
- Initial release of mitm-gui
- Fyne-based desktop GUI with Settings and Workflow tabs
- AES-256-GCM encryption with Argon2id key derivation for INI configuration files
- Encrypted INI loader supporting GLOBAL, HR, ORG, GEO topic sections
- Multi-format data ingestion:
  - CSV parser (Go encoding/csv)
  - Excel parser (excelize/v2)
  - Kafka consumer (segmentio/kafka-go) with SASL/SSL and TLS
- JSON Schema validation (santhosh-tekuri/jsonschema/v5) per topic
- HTTP upload with two modes:
  - Direct POST with optional proxy
  - Authenticated upload (refresh token -> access token -> Bearer upload)
- Per-user encrypted proxy configuration (proxy_<user>.enc)
- Dual logging system: file output + GUI summary panel
- Per-topic audit logging (DATA_LOAD, VALIDATE, UPLOAD_START, UPLOAD_RESULT)
- Workflow state machine enforcing valid transitions
- Thread-safe GUI updates with Fyne
- Kafka data flattening:EmploymentInformation extraction + JobInformation promotion
- CLI mode for configuration dump and testing
- encryptini CLI tool for encrypting plaintext INI files
- Build script (build.bat) with tidy/build/test/all/cli/clean targets
- 30+ tests across all packages
