// Main entry point for mitm-gui application.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2/app"

	"github.com/zheng-bote/go_mitm-gui/gui"
	"github.com/zheng-bote/go_mitm-gui/internal/config"
)

func main() {
	cliMode := flag.Bool("cli", false, "Run in CLI mode instead of GUI")
	encryptedPath := flag.String("config", "", "Path to encrypted INI file (CLI mode)")
	password := flag.String("password", "", "Master password to decrypt config (CLI mode)")
	flag.Parse()

	if *cliMode {
		runCLI(*encryptedPath, *password)
		return
	}
	runGUI()
}

func runGUI() {
	exe, _ := os.Executable()
	logDir := filepath.Dir(exe)
	logPath := filepath.Join(logDir, "mitm-gui.log")

	a := app.NewWithID("com.zheng-bote.mitm-gui")
	w := a.NewWindow("mitm-gui — Configuration Loader")
	w.SetMaster()

	_ = gui.NewApp(w, logPath)
	w.ShowAndRun()
}

func runCLI(encryptedPath, password string) {
	if encryptedPath == "" || password == "" {
		fmt.Fprintln(os.Stderr, `Usage: mitm-gui -cli -config=<file.enc> -password=<pwd>`)
		os.Exit(1)
	}
	loader := config.New()
	appCfg, err := loader.LoadConfig(encryptedPath, password)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Println("=== mitm-gui Configuration ===")
	fmt.Printf("Global:\n")
	fmt.Printf("  Admins:     %v\n", appCfg.Global.Admins)
	fmt.Printf("  Log Level:  %s\n", appCfg.Global.LogLevel)
	fmt.Printf("  Log File:   %s\n", appCfg.Global.LogFilePath)

	for name, topic := range appCfg.Topics {
		fmt.Printf("\n[%s]\n", name)
		fmt.Printf("  Excel:       %t\n", topic.InputExcel)
		fmt.Printf("  CSV:         %t\n", topic.InputCSV)
		fmt.Printf("  Oracle DB:   %t\n", topic.InputOraDB)
		fmt.Printf("  Postgres DB: %t\n", topic.InputPGDB)
		fmt.Printf("  Kafka:       %t\n", topic.InputKafka)
		fmt.Printf("  Schema:      %s\n", topic.JSONSchemaPath)
		fmt.Printf("  Upload URL:  %s\n", topic.UploadURL)
		fmt.Printf("  Audit Log:   %s (enabled: %t)\n",
			topic.AuditLogFilePath, topic.AuditLogEnabled)
		if topic.ExcelInput != nil {
			fmt.Printf("  Excel Path:  %s\n", topic.ExcelInput.DefaultFilePath)
		}
		if topic.CSVInput != nil {
			fmt.Printf("  CSV Path:    %s\n", topic.CSVInput.DefaultFilePath)
		}
		if topic.OraDBInput != nil {
			fmt.Printf("  Oracle DB:   %s@%s:%d/%s\n",
				topic.OraDBInput.Username, topic.OraDBInput.Host,
				topic.OraDBInput.Port, topic.OraDBInput.ServiceName)
		}
		if topic.KafkaInput != nil {
			fmt.Printf("  Kafka:       %s / %s\n",
				topic.KafkaInput.BootstrapServers, topic.KafkaInput.Topic)
		}
	}
}
