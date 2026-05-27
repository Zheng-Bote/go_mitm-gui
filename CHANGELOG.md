# Changelog

## [0.2.0] - 2026-05-27

### Added
- **Kafka Debug Mode:** Optional export of loaded Kafka records to `<yyyy-mm-dd>_json-upload.json` (enabled via `kafka_debug_mode=true` in INI).
- **Dynamic Kafka GroupID:** Automatic appending of OS username and timestamp to `GROUP_ID` if it starts with `bmw.cority.connect.`.
- **Proxy Auto-Detection:** Settings tab now automatically detects `proxy_<user>.enc` in the INI directory and prompts for password.
- **Background Processing:** Data loading and validation now run in separate goroutines to prevent GUI freezes.

### Optimized
- **Parallel Validation:** JSON Schema validation now uses multiple CPU cores for significant speedups on large datasets.
- **Memory Efficiency:** Upload pipeline refactored to use `json.RawMessage`, drastically reducing memory usage and allocations for large payloads.
- **GUI Stability:** Implementation of log truncation (last 100 entries) in the summary panel to maintain responsiveness with massive error lists.

### Fixed
- **Kafka Timeout Detection:** Improved robust detection of wrapped `context.DeadlineExceeded` errors using `errors.Is`.
- Corrected misplaced imports in `internal/model/types.go`.
- Fixed undefined icon error in `gui/settings.go` by switching to `theme.FolderOpenIcon`.

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
