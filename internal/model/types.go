// Package model defines the core data structures for the mitm-gui application.
package model

import "encoding/json"

type AppConfig struct {
	Global GlobalConfig
	Topics map[string]*TopicConfig
}

type GlobalConfig struct {
	Admins      []string
	LogLevel    string
	LogFilePath string
}

type TopicConfig struct {
	InputExcel bool
	InputCSV   bool
	InputOraDB bool
	InputPGDB  bool
	InputKafka bool

	AuditLogEnabled  bool
	AuditLogFilePath string

	JSONSchemaPath string
	UploadURL      string

	ExcelInput *ExcelInputConfig
	CSVInput   *CSVInputConfig
	OraDBInput *OraDBInputConfig
	PGDBInput  *PGDBInputConfig
	KafkaInput *KafkaInputConfig

	Auth           *AuthConfig
	UploadEndpoint *UploadEndpointConfig
}

type ExcelInputConfig struct {
	DefaultFilePath string `ini:"DEFAULT_FILE_PATH"`
}

type CSVInputConfig struct {
	DefaultFilePath string `ini:"DEFAULT_FILE_PATH"`
}

type OraDBInputConfig struct {
	Host        string `ini:"HOST"`
	Port        int    `ini:"PORT"`
	ServiceName string `ini:"SERVICE_NAME"`
	Username    string `ini:"USERNAME"`
	Password    string `ini:"PASSWORD"`
}

type PGDBInputConfig struct {
	Host        string `ini:"HOST"`
	Port        int    `ini:"PORT"`
	ServiceName string `ini:"SERVICE_NAME"`
	Username    string `ini:"USERNAME"`
	Password    string `ini:"PASSWORD"`
}

type KafkaInputConfig struct {
	BootstrapServers string `ini:"BOOTSTRAP_SERVERS"`
	Topic            string `ini:"TOPIC"`
	GroupID          string `ini:"GROUP_ID"`
	Key              string `ini:"KEY"`
	Secret           string `ini:"SECRET"`
	SecurityProtocol string `ini:"security_protocol"`
	SASLMechanism    string `ini:"sasl_mechanism"`
	AutoOffsetReset  string `ini:"auto_offset_reset"`
	ConsumerTimeout  int    `ini:"consumer_timeout"`
	KafkaDebugMode   bool   `ini:"kafka_debug_mode"`
}

type ProxyConfig struct {
	Server   string `ini:"server"`
	Port     int    `ini:"port"`
	User     string `ini:"user"`
	Password string `ini:"password"`
}

type AuthConfig struct {
	BaseURL  string `ini:"saas_base_url"`
	Login    string `ini:"saas_login"`
	Password string `ini:"saas_password"`
}

type UploadEndpointConfig struct {
	UseProxy  bool   `ini:"use_proxy"`
	ProxyHost string `ini:"proxy_host"`
	UploadURL string `ini:"upload_url"`
	Timeout   int    `ini:"upload_timeout"`
}

// --- Sprint 4+ types ---

type DataFormat string

const (
	FormatCSV   DataFormat = "CSV"
	FormatExcel DataFormat = "Excel"
	FormatOraDB DataFormat = "Oracle DB"
	FormatPGDB  DataFormat = "PostgreSQL"
	FormatKafka DataFormat = "Kafka"
)

type LoadedData struct {
	Format      DataFormat
	SourcePath  string
	RowCount    int
	JSONPayload []byte
	Fields      []string
}

type ValidationResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
}

type UploadResult struct {
	Success      bool
	StatusCode   int
	ResponseBody string
	Error        string
}

type WorkflowStep int

const (
	StepIdle       WorkflowStep = iota
	StepDataLoaded
	StepValidated
	StepUploaded
	StepError
)

type UploadOptions struct {
	UpdateExistingRecords       string `json:"updateExistingRecords"`
	InsertBaseTables            string `json:"insertBaseTables"`
	ForceLookupTableUpdate      string `json:"forceLookupTableUpdate"`
	DisableSegUpdate            string `json:"disableSegUpdate"`
	AutoCreatePortalUser        string `json:"autoCreatePortalUser"`
	MergeRecordsWithMatchingSsn string `json:"mergeRecordsWithMatchingSsn"`
	DateFormat                  string `json:"dateFormat"`
}

func DefaultUploadOptions() UploadOptions {
	return UploadOptions{
		UpdateExistingRecords:       "true",
		InsertBaseTables:            "true",
		ForceLookupTableUpdate:      "true",
		DisableSegUpdate:            "false",
		AutoCreatePortalUser:        "true",
		MergeRecordsWithMatchingSsn: "false",
		DateFormat:                  "dd.mm.yyyy",
	}
}

type UploadEnvelope struct {
	Options UploadOptions   `json:"options"`
	Records json.RawMessage `json:"records"`
}
