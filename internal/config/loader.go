// Package config provides loading and parsing of encrypted INI configuration files.
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/ini.v1"

	"github.com/zheng-bote/go_mitm-gui/internal/crypto"
	"github.com/zheng-bote/go_mitm-gui/internal/model"
)

// Loader handles decryption and parsing of the encrypted INI configuration.
type Loader struct{}

// New creates a new Loader.
func New() *Loader {
	return &Loader{}
}

// LoadConfig decrypts an encrypted INI file and parses it into an AppConfig.
func (l *Loader) LoadConfig(encryptedPath, masterPassword string) (*model.AppConfig, error) {
	// Read the encrypted file.
	encryptedData, err := os.ReadFile(encryptedPath)
	if err != nil {
		return nil, fmt.Errorf("config: failed to read encrypted file %q: %w", encryptedPath, err)
	}

	// Decrypt.
	plaintext, err := crypto.Decrypt(masterPassword, encryptedData)
	if err != nil {
		return nil, fmt.Errorf("config: failed to decrypt %q: %w", encryptedPath, err)
	}

	// Parse the INI content.
	return parseINIContent(plaintext)
}

// LoadPlainConfig parses an unencrypted INI file content directly.
// Useful for testing or development.
func (l *Loader) LoadPlainConfig(plaintext []byte) (*model.AppConfig, error) {
	return parseINIContent(plaintext)
}

// parseINIContent parses raw INI bytes into an AppConfig.
func parseINIContent(data []byte) (*model.AppConfig, error) {
	cfg, err := ini.Load(data)
	if err != nil {
		return nil, fmt.Errorf("config: failed to parse INI: %w", err)
	}

	appCfg := &model.AppConfig{
		Topics: make(map[string]*model.TopicConfig),
	}

	// Parse [GLOBAL].
	if err := parseGlobal(cfg, appCfg); err != nil {
		return nil, err
	}

	// Determine which topic sections exist by looking for known section names.
	topicNames := []string{"HR", "ORG", "GEO"}
	for _, name := range topicNames {
		if cfg.Section(name) != nil && cfg.Section(name).HasKey("INPUT_EXCEL") {
			topicCfg, err := parseTopic(cfg, name)
			if err != nil {
				return nil, fmt.Errorf("config: failed to parse topic [%s]: %w", name, err)
			}
			appCfg.Topics[name] = topicCfg
		}
	}

	return appCfg, nil
}

// parseGlobal reads the [GLOBAL] section.
func parseGlobal(cfg *ini.File, appCfg *model.AppConfig) error {
	sec, err := cfg.GetSection("GLOBAL")
	if err != nil {
		return fmt.Errorf("config: missing [GLOBAL] section: %w", err)
	}

	appCfg.Global.LogLevel = sec.Key("log_level").MustString("INFO")

	appCfg.Global.LogFilePath = sec.Key("log_file_path").String()

	// Parse Admins list: "q225265", "zb_bamboo"
	adminsRaw := sec.Key("Admins").String()
	if adminsRaw != "" {
		// Remove surrounding quotes and split by comma.
		adminsRaw = strings.TrimSpace(adminsRaw)
		adminsRaw = strings.Trim(adminsRaw, `"`)
		parts := strings.Split(adminsRaw, `", "`)
		for _, p := range parts {
			name := strings.Trim(p, `" `)
			if name != "" {
				appCfg.Global.Admins = append(appCfg.Global.Admins, name)
			}
		}
	}

	return nil
}

// parseTopic reads a topic section (e.g., [HR]) and its sub-sections.
func parseTopic(cfg *ini.File, topicName string) (*model.TopicConfig, error) {
	sec, err := cfg.GetSection(topicName)
	if err != nil {
		return nil, err
	}

	tc := &model.TopicConfig{}

	// Format flags.
	tc.InputExcel = sec.Key("INPUT_EXCEL").MustBool(false)
	tc.InputCSV = sec.Key("INPUT_CSV").MustBool(false)
	tc.InputOraDB = sec.Key("INPUT_ORA-DB").MustBool(false)
	tc.InputPGDB = sec.Key("INPUT_PG-DB").MustBool(false)
	tc.InputKafka = sec.Key("INPUT_KAFKA").MustBool(false)

	// Audit log.
	tc.AuditLogEnabled = sec.Key("audit_log_enabled").MustBool(false)
	tc.AuditLogFilePath = sec.Key("audit_log_file_path").String()

	// Schema and upload URL.
	tc.JSONSchemaPath = sec.Key("JSON_SCHEMA_PATH").String()
	tc.UploadURL = sec.Key("UPLOAD_URL").String()

	// Parse sub-sections for each enabled format.
	if tc.InputExcel {
		tc.ExcelInput, _ = parseExcelInput(cfg, topicName)
	}
	if tc.InputCSV {
		tc.CSVInput, _ = parseCSVInput(cfg, topicName)
	}
	if tc.InputOraDB {
		tc.OraDBInput, _ = parseOraDBInput(cfg, topicName)
	}
	if tc.InputPGDB {
		tc.PGDBInput, _ = parsePGDBInput(cfg, topicName)
	}
	if tc.InputKafka {
		tc.KafkaInput, _ = parseKafkaInput(cfg, topicName)
	}

	// Parse optional [TOPIC-AUTH] section.
	tc.Auth, _ = parseAuthConfig(cfg, topicName)

	// Parse optional [TOPIC-UPLOAD] section.
	tc.UploadEndpoint, _ = parseUploadEndpointConfig(cfg, topicName)

	return tc, nil
}

// subSectionName builds the INI sub-section name for a topic and format.
// e.g., "HR", "INPUT-EXCEL" -> "HR-INPUT-EXCEL"
func subSectionName(topicName, formatSuffix string) string {
	return topicName + "-INPUT-" + formatSuffix
}

// parseExcelInput reads the [TOPIC-INPUT-EXCEL] sub-section.
func parseExcelInput(cfg *ini.File, topicName string) (*model.ExcelInputConfig, error) {
	secName := subSectionName(topicName, "EXCEL")
	sec, err := cfg.GetSection(secName)
	if err != nil {
		return nil, err
	}
	return &model.ExcelInputConfig{
		DefaultFilePath: sec.Key("DEFAULT_FILE_PATH").String(),
	}, nil
}

// parseCSVInput reads the [TOPIC-INPUT-CSV] sub-section.
func parseCSVInput(cfg *ini.File, topicName string) (*model.CSVInputConfig, error) {
	secName := subSectionName(topicName, "CSV")
	sec, err := cfg.GetSection(secName)
	if err != nil {
		return nil, err
	}
	return &model.CSVInputConfig{
		DefaultFilePath: sec.Key("DEFAULT_FILE_PATH").String(),
	}, nil
}

// parseOraDBInput reads the [TOPIC-INPUT-ORA-DB] sub-section.
func parseOraDBInput(cfg *ini.File, topicName string) (*model.OraDBInputConfig, error) {
	secName := subSectionName(topicName, "ORA-DB")
	sec, err := cfg.GetSection(secName)
	if err != nil {
		return nil, err
	}
	return &model.OraDBInputConfig{
		Host:        sec.Key("HOST").String(),
		Port:        sec.Key("PORT").MustInt(1521),
		ServiceName: sec.Key("SERVICE_NAME").String(),
		Username:    sec.Key("USERNAME").String(),
		Password:    sec.Key("PASSWORD").String(),
	}, nil
}

// parsePGDBInput reads the [TOPIC-INPUT-PG-DB] sub-section.
func parsePGDBInput(cfg *ini.File, topicName string) (*model.PGDBInputConfig, error) {
	secName := subSectionName(topicName, "PG-DB")
	sec, err := cfg.GetSection(secName)
	if err != nil {
		return nil, err
	}
	return &model.PGDBInputConfig{
		Host:        sec.Key("HOST").String(),
		Port:        sec.Key("PORT").MustInt(5432),
		ServiceName: sec.Key("SERVICE_NAME").String(),
		Username:    sec.Key("USERNAME").String(),
		Password:    sec.Key("PASSWORD").String(),
	}, nil
}

// parseKafkaInput reads the [TOPIC-INPUT-KAFKA] sub-section.
// parseAuthConfig reads the [TOPIC-AUTH] sub-section (optional).
func parseAuthConfig(cfg *ini.File, topicName string) (*model.AuthConfig, error) {
	secName := topicName + "-AUTH"
	sec, err := cfg.GetSection(secName)
	if err != nil {
		return nil, err
	}
	return &model.AuthConfig{
		BaseURL:  sec.Key("saas_base_url").String(),
		Login:    sec.Key("saas_login").String(),
		Password: sec.Key("saas_password").String(),
	}, nil
}

// parseUploadEndpointConfig reads the [TOPIC-UPLOAD] sub-section (optional).
func parseUploadEndpointConfig(cfg *ini.File, topicName string) (*model.UploadEndpointConfig, error) {
	secName := topicName + "-UPLOAD"
	sec, err := cfg.GetSection(secName)
	if err != nil {
		return nil, err
	}
	return &model.UploadEndpointConfig{
		UseProxy:  sec.Key("use_proxy").MustBool(false),
		ProxyHost: sec.Key("proxy_host").String(),
		UploadURL: sec.Key("upload_url").String(),
		Timeout:   sec.Key("upload_timeout").MustInt(120),
	}, nil
}

func parseKafkaInput(cfg *ini.File, topicName string) (*model.KafkaInputConfig, error) {
	secName := subSectionName(topicName, "KAFKA")
	sec, err := cfg.GetSection(secName)
	if err != nil {
		return nil, err
	}
	return &model.KafkaInputConfig{
		BootstrapServers: sec.Key("BOOTSTRAP_SERVERS").String(),
		Topic:            sec.Key("TOPIC").String(),
		GroupID:          sec.Key("GROUP_ID").String(),
		Key:              sec.Key("KEY").String(),
		Secret:           sec.Key("SECRET").String(),
		SecurityProtocol: sec.Key("security_protocol").String(),
		SASLMechanism:    sec.Key("sasl_mechanism").String(),
		AutoOffsetReset:  sec.Key("auto_offset_reset").String(),
		ConsumerTimeout:  sec.Key("consumer_timeout").MustInt(30),
	}, nil
}
