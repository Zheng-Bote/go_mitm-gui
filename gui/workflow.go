package gui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/zheng-bote/go_mitm-gui/internal/input"
	"github.com/zheng-bote/go_mitm-gui/internal/logging"
	"github.com/zheng-bote/go_mitm-gui/internal/model"
	"github.com/zheng-bote/go_mitm-gui/internal/schema"
	"github.com/zheng-bote/go_mitm-gui/internal/upload"
	"github.com/zheng-bote/go_mitm-gui/internal/workflow"
)

type WorkflowTab struct {
	parent      fyne.Window
	config      *model.AppConfig
	proxyConfig *model.ProxyConfig
	log         *logging.Logger
	ctrl        *workflow.Controller

	currentTopic string
	currentFmt   model.DataFormat
	currentData  *model.LoadedData

	configLabel  *widget.Label
	topicSelect  *widget.Select
	formatSelect *widget.Select

	filePathLabel *widget.Label
	filePath      string
	selectFileBtn *widget.Button

	loadDataBtn *widget.Button
	validateBtn *widget.Button
	uploadBtn   *widget.Button

	progress    *widget.ProgressBar
	statusLabel *widget.Label
	stepLabels  map[string]*widget.Label

	summary *widget.Entry
	mu      sync.Mutex
}

func NewWorkflowTab(parent fyne.Window, logger *logging.Logger) *WorkflowTab {
	wt := &WorkflowTab{
		parent:     parent,
		log:        logger,
		stepLabels: make(map[string]*widget.Label),
	}
	wt.ctrl = workflow.NewController(logger)

	wt.configLabel = widget.NewLabel("No configuration loaded.")
	wt.topicSelect = widget.NewSelect([]string{}, func(topic string) {
		wt.currentTopic = topic
		wt.onTopicChanged(topic)
	})
	wt.topicSelect.PlaceHolder = "Select topic..."
	wt.topicSelect.Disable()

	wt.formatSelect = widget.NewSelect([]string{}, func(fmtStr string) {
		wt.currentFmt = model.DataFormat(fmtStr)
		wt.onFormatChanged()
	})
	wt.formatSelect.PlaceHolder = "Select format..."
	wt.formatSelect.Disable()

	wt.filePathLabel = widget.NewLabel("(no file selected)")
	wt.selectFileBtn = widget.NewButton("Select Input File...", wt.onSelectFile)
	wt.selectFileBtn.Disable()

	wt.stepLabels["input"] = newStepLabel("Data Input:  ", false)
	wt.stepLabels["load"] = newStepLabel("Data Load:   ", false)
	wt.stepLabels["validate"] = newStepLabel("Validation:  ", false)
	wt.stepLabels["upload"] = newStepLabel("Upload:      ", false)

	wt.statusLabel = widget.NewLabel("")
	wt.progress = widget.NewProgressBar()
	wt.progress.Hide()

	wt.summary = widget.NewMultiLineEntry()
	wt.summary.SetMinRowsVisible(8)
	wt.summary.Wrapping = fyne.TextWrapWord

	logger.SetCallback(func(level logging.Level, ts time.Time, msg string) {
		prefix := ""
		switch level {
		case logging.LevelWarn:
			prefix = "[WARN] "
		case logging.LevelError:
			prefix = "[ERROR] "
		default:
			prefix = "[INFO] "
		}
		wt.appendSummary(prefix + msg)
	})

	wt.ctrl.SetStateChangeCallback(func(from, to workflow.State) {
		wt.mu.Lock()
		wt.loadDataBtn.Disable()
		wt.validateBtn.Disable()
		wt.uploadBtn.Disable()
		wt.mu.Unlock()

		if wt.ctrl.CanLoad() && wt.filePath != "" {
			wt.loadDataBtn.Enable()
		}
		if wt.ctrl.CanValidate() {
			wt.validateBtn.Enable()
		}
		if wt.ctrl.CanUpload() {
			wt.uploadBtn.Enable()
		}
	})

	wt.loadDataBtn = widget.NewButtonWithIcon("Load Data", theme.DownloadIcon(), wt.onLoadData)
	wt.loadDataBtn.Importance = widget.HighImportance
	wt.loadDataBtn.Disable()

	wt.validateBtn = widget.NewButtonWithIcon("Validate", theme.VisibilityIcon(), wt.onValidate)
	wt.validateBtn.Disable()

	wt.uploadBtn = widget.NewButtonWithIcon("Upload", theme.UploadIcon(), wt.onUpload)
	wt.uploadBtn.Disable()

	return wt
}

func (wt *WorkflowTab) SetProxyConfig(pc *model.ProxyConfig) {
	wt.mu.Lock()
	wt.proxyConfig = pc
	wt.mu.Unlock()
}

func newStepLabel(prefix string, ok bool) *widget.Label {
	t := prefix + "pending"
	if ok {
		t = prefix + "done"
	}
	return widget.NewLabel(t)
}

func (wt *WorkflowTab) Build() fyne.CanvasObject {
	selectorRow := container.NewGridWithColumns(2,
		container.NewVBox(
			widget.NewLabelWithStyle("Topic", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			wt.topicSelect,
		),
		container.NewVBox(
			widget.NewLabelWithStyle("Input Format", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			wt.formatSelect,
		),
	)
	fileSection := container.NewVBox(
		widget.NewSeparator(),
		wt.selectFileBtn,
		container.NewHBox(widget.NewLabel("File:"), wt.filePathLabel),
		widget.NewSeparator(),
	)
	btnRow := container.NewHBox(wt.loadDataBtn, wt.validateBtn, wt.uploadBtn)
	stepBox := container.NewVBox()
	for _, k := range []string{"input", "load", "validate", "upload"} {
		stepBox.Add(wt.stepLabels[k])
	}
	progressBox := container.NewVBox(wt.progress, wt.statusLabel)
	summaryLabel := widget.NewLabelWithStyle("Summary / Errors / Warnings",
		fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	summaryScroll := container.NewScroll(wt.summary)
	summaryScroll.SetMinSize(fyne.NewSize(760, 200))

	return container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Workflow", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			wt.configLabel,
			widget.NewSeparator(),
			selectorRow,
			fileSection,
			btnRow,
		),
		nil, nil, nil,
		container.NewVSplit(
			container.NewVBox(widget.NewSeparator(), stepBox, progressBox),
			container.NewVBox(widget.NewSeparator(), summaryLabel, summaryScroll),
		),
	)
}

func (wt *WorkflowTab) SetConfig(cfg *model.AppConfig) {
	wt.mu.Lock()
	wt.config = cfg
	wt.mu.Unlock()

	wt.ctrl.SetConfig(cfg)
	wt.configLabel.SetText(fmt.Sprintf("Configuration loaded: %d topic(s)", len(cfg.Topics)))
	topics := make([]string, 0, len(cfg.Topics))
	for name := range cfg.Topics {
		topics = append(topics, name)
	}
	wt.topicSelect.Options = topics
	wt.topicSelect.Enable()
	wt.topicSelect.SetSelected("")
	wt.resetWorkflow()
}

func (wt *WorkflowTab) onTopicChanged(topic string) {
	tc := wt.currentTopicConfig()
	if tc == nil {
		return
	}
	var formats []string
	if tc.InputCSV {
		formats = append(formats, string(model.FormatCSV))
	}
	if tc.InputExcel {
		formats = append(formats, string(model.FormatExcel))
	}
	if tc.InputOraDB {
		formats = append(formats, string(model.FormatOraDB))
	}
	if tc.InputPGDB {
		formats = append(formats, string(model.FormatPGDB))
	}
	if tc.InputKafka {
		formats = append(formats, string(model.FormatKafka))
	}
	wt.formatSelect.Options = formats
	if len(formats) > 0 {
		wt.formatSelect.PlaceHolder = "Select format..."
		wt.formatSelect.Enable()
	} else {
		wt.formatSelect.PlaceHolder = "No formats available"
		wt.formatSelect.Disable()
	}
	wt.formatSelect.SetSelected("")
	wt.resetWorkflow()
}

func (wt *WorkflowTab) onFormatChanged() {
	wt.currentData = nil
	switch wt.currentFmt {
	case model.FormatCSV, model.FormatExcel:
		wt.selectFileBtn.Enable()
		wt.filePath = ""
		wt.filePathLabel.SetText("(select a file...)")
		wt.loadDataBtn.Disable()
		wt.setStep("input", true)
		wt.setStep("load", false)
		wt.setStep("validate", false)
		wt.setStep("upload", false)
		wt.statusLabel.SetText("Select an input file.")
	case model.FormatKafka:
		wt.selectFileBtn.Disable()
		wt.filePath = "kafka://" + wt.currentTopic
		wt.filePathLabel.SetText("Kafka topic (no file needed)")
		wt.loadDataBtn.Enable()
		wt.setStep("input", true)
		wt.setStep("load", true)
		wt.statusLabel.SetText("Ready to load from Kafka. Click Load Data.")
	default:
		wt.selectFileBtn.Disable()
		wt.filePathLabel.SetText(fmt.Sprintf("Format %s is not yet implemented.", wt.currentFmt))
	}
}

func (wt *WorkflowTab) onSelectFile() {
	var filter storage.FileFilter
	switch wt.currentFmt {
	case model.FormatCSV:
		filter = storage.NewExtensionFileFilter([]string{".csv"})
	case model.FormatExcel:
		filter = storage.NewExtensionFileFilter([]string{".xlsx", ".xls"})
	default:
		return
	}

	// Reset workflow on re-selection if data was already loaded.
	if wt.currentData != nil {
		wt.ctrl.ResetToConfig()
		wt.resetWorkflow()
		wt.formatSelect.SetSelected("")
	}

	dlg := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(fmt.Errorf("Error opening file: %w", err), wt.parent)
			return
		}
		if reader == nil {
			return
		}
		wt.filePath = reader.URI().Path()
		wt.filePathLabel.SetText(wt.filePath)
		if wt.ctrl.CanLoad() {
			wt.loadDataBtn.Enable()
		}
		wt.statusLabel.SetText("File selected. Click Load Data to parse.")
		_ = reader.Close()
	}, wt.parent)
	dlg.SetFilter(filter)
	dlg.Show()
}

func (wt *WorkflowTab) onLoadData() {
	if wt.filePath == "" && wt.currentFmt != model.FormatKafka {
		dialog.ShowInformation("No File", "Please select a file first.", wt.parent)
		return
	}
	wt.log.Info("Loading %s file: %s", wt.currentFmt, wt.filePath)
	wt.audit("DATA_LOAD", fmt.Sprintf("%s: %s", wt.currentFmt, wt.filePath))
	wt.setStep("input", true)
	wt.setStep("load", true)
	wt.progress.SetValue(0.1)
	wt.statusLabel.SetText("Loading data...")

	go func() {
		source := input.NewDataSource(wt.currentFmt, wt.currentTopicConfig())
		if source == nil {
			dialog.ShowError(fmt.Errorf("Format %s is not yet implemented", wt.currentFmt), wt.parent)
			return
		}

		data, err := source.Load(wt.filePath)
		if err != nil {
			wt.log.Error("Data load failed: %v", err)
			dialog.ShowError(fmt.Errorf("Data load failed: %w", err), wt.parent)
			wt.setStep("load", false)
			wt.progress.SetValue(0)
			wt.statusLabel.SetText("Data load failed.")
			return
		}

		wt.mu.Lock()
		wt.currentData = data
		wt.mu.Unlock()

		if err := wt.ctrl.SetData(data); err != nil {
			wt.log.Error("Controller rejected data: %v", err)
			return
		}

		wt.progress.SetValue(0.4)
		wt.statusLabel.SetText(fmt.Sprintf("Loaded %d rows from %s.", data.RowCount, data.SourcePath))
		wt.formatSelect.SetSelected("")

		preview := string(data.JSONPayload)
		if len(preview) > 500 {
			preview = preview[:500] + "\n... (truncated)"
		}
		wt.appendSummary(fmt.Sprintf("Loaded %d rows. Fields: %v\nPreview:\n%s",
			data.RowCount, data.Fields, preview))
	}()
}

func (wt *WorkflowTab) onValidate() {
	if wt.currentData == nil {
		dialog.ShowInformation("No Data", "Load data first.", wt.parent)
		return
	}
	tc := wt.currentTopicConfig()
	if tc == nil {
		dialog.ShowInformation("No Topic", "Select a topic first.", wt.parent)
		return
	}
	if tc.JSONSchemaPath == "" {
		dialog.ShowInformation("No Schema", "This topic has no JSON Schema.", wt.parent)
		return
	}

	wt.setStep("validate", true)
	wt.progress.SetValue(0.5)
	wt.statusLabel.SetText("Validating...")
	wt.log.Info("Validating %d rows against %q", wt.currentData.RowCount, tc.JSONSchemaPath)
	wt.audit("VALIDATE", fmt.Sprintf("schema: %s, rows: %d", tc.JSONSchemaPath, wt.currentData.RowCount))

	go func() {
		validator := schema.New()
		result, err := validator.ValidateLoaded(wt.currentData, tc.JSONSchemaPath)
		if err != nil {
			wt.log.Error("Validation failed: %v", err)
			dialog.ShowError(fmt.Errorf("Validation failed: %w", err), wt.parent)
			wt.setStep("validate", false)
			wt.progress.SetValue(0)
			wt.statusLabel.SetText("Validation failed.")
			return
		}

		wt.ctrl.SetValidation(result)

		if result.Valid {
			wt.progress.SetValue(0.7)
			wt.statusLabel.SetText("Validation passed.")
		} else {
			wt.progress.SetValue(0.5)
			wt.statusLabel.SetText("Validation found errors.")
		}

		summary := fmt.Sprintf("Validation result: %s",
			map[bool]string{true: "PASSED", false: "FAILED"}[result.Valid])
		if len(result.Errors) > 0 {
			summary += "\nErrors:\n  " + strings.Join(result.Errors, "\n  ")
		}
		if len(result.Warnings) > 0 {
			summary += "\nWarnings:\n  " + strings.Join(result.Warnings, "\n  ")
		}
		wt.appendSummary(summary)
	}()
}

func (wt *WorkflowTab) onUpload() {
	if wt.currentData == nil {
		dialog.ShowInformation("No Data", "Load data first.", wt.parent)
		return
	}
	tc := wt.currentTopicConfig()
	if tc == nil {
		dialog.ShowInformation("No Topic", "Select a topic first.", wt.parent)
		return
	}

	uploadURL := tc.UploadURL
	if tc.UploadEndpoint != nil && tc.UploadEndpoint.UploadURL != "" {
		uploadURL = tc.UploadEndpoint.UploadURL
	}
	if uploadURL == "" {
		dialog.ShowInformation("No Upload URL", "This topic has no upload URL.", wt.parent)
		return
	}

	wt.setStep("upload", true)
	wt.progress.SetValue(0.8)
	wt.statusLabel.SetText("Uploading...")
	wt.log.Info("Uploading %d records to %q", wt.currentData.RowCount, uploadURL)
	wt.audit("UPLOAD_START", fmt.Sprintf("url: %s, rows: %d", uploadURL, wt.currentData.RowCount))

	// Check if proxy should be used.
	useProxy := (tc.UploadEndpoint != nil && tc.UploadEndpoint.UseProxy) ||
		(wt.proxyConfig != nil && wt.proxyConfig.Server != "")
	if useProxy {
		if wt.proxyConfig != nil && wt.proxyConfig.Server != "" {
			wt.log.Info("Using internet proxy: %s:%d (user: %s)",
				wt.proxyConfig.Server, wt.proxyConfig.Port, wt.proxyConfig.User)
			wt.appendSummary(fmt.Sprintf("Proxy: %s:%d", wt.proxyConfig.Server, wt.proxyConfig.Port))
		} else {
			wt.log.Warn("use_proxy=true but no proxy config loaded")
		}
	} else {
		wt.log.Info("No proxy configured - direct connection")
	}

	go func() {
		var result *model.UploadResult
		if tc.Auth != nil && tc.Auth.BaseURL != "" {
			result = upload.AuthenticatedUpload(wt.currentData.JSONPayload, uploadURL, tc.Auth, nil, wt.proxyConfig)
		} else {
			client := upload.NewClient()
			if useProxy && wt.proxyConfig != nil && wt.proxyConfig.Server != "" {
				client = upload.NewClientWithProxy(wt.proxyConfig)
			}
			var err error
			result, err = client.Upload(wt.currentData.JSONPayload, uploadURL, nil)
			if err != nil {
				result = &model.UploadResult{Success: false, Error: err.Error()}
			}
		}
		if result == nil {
			wt.setStep("upload", false)
			wt.progress.SetValue(0)
			wt.statusLabel.SetText("Upload failed.")
			return
		}
		wt.ctrl.SetUpload(result)
		wt.audit("UPLOAD_RESULT", fmt.Sprintf("HTTP %d: %s", result.StatusCode, result.ResponseBody))
		if result.Success {
			wt.progress.SetValue(1.0)
			wt.statusLabel.SetText("Upload completed.")
		} else {
			wt.progress.SetValue(0.8)
			wt.statusLabel.SetText("Upload error.")
		}
		summary := fmt.Sprintf("Upload: %s (HTTP %d)\nBody: %s",
			map[bool]string{true: "SUCCESS", false: "FAILED"}[result.Success],
			result.StatusCode, result.ResponseBody)
		if result.Error != "" {
			summary += "\nError: " + result.Error
		}
		wt.appendSummary(summary)
	}()
}

func (wt *WorkflowTab) resetWorkflow() {
	wt.currentData = nil
	wt.currentFmt = ""
	wt.filePath = ""
	wt.filePathLabel.SetText("(no file selected)")
	wt.selectFileBtn.Disable()
	wt.loadDataBtn.Disable()
	wt.validateBtn.Disable()
	wt.uploadBtn.Disable()
	wt.progress.SetValue(0)
	wt.progress.Hide()
	wt.statusLabel.SetText("")
	wt.summary.SetText("")
	for k := range wt.stepLabels {
		wt.setStep(k, false)
	}
}

func (wt *WorkflowTab) setStep(name string, done bool) {
	lbl, ok := wt.stepLabels[name]
	if !ok {
		return
	}
	var base string
	switch name {
	case "input":
		base = "Data Input:  "
	case "load":
		base = "Data Load:   "
	case "validate":
		base = "Validation:  "
	case "upload":
		base = "Upload:      "
	}
	if done {
		lbl.SetText(base + "done")
	} else {
		lbl.SetText(base + "pending")
	}
}

func (wt *WorkflowTab) appendSummary(text string) {
	wt.mu.Lock()
	defer wt.mu.Unlock()

	lines := strings.Split(wt.summary.Text, "\n---\n")
	if len(lines) > 100 {
		lines = lines[len(lines)-100:]
		wt.summary.SetText("(truncated for performance... check audit logs for full details)\n---\n" + strings.Join(lines, "\n---\n") + "\n---\n" + text)
		return
	}

	current := wt.summary.Text
	if current == "" {
		wt.summary.SetText(text)
	} else {
		wt.summary.SetText(current + "\n---\n" + text)
	}
}

func (wt *WorkflowTab) audit(event, details string) {
	tc := wt.currentTopicConfig()
	if tc == nil {
		return
	}
	al := logging.NewAuditLogger(tc.AuditLogFilePath, tc.AuditLogEnabled)
	al.Log(event, details)
}

func (wt *WorkflowTab) currentTopicConfig() *model.TopicConfig {
	if wt.config == nil || wt.currentTopic == "" {
		return nil
	}
	return wt.config.Topics[wt.currentTopic]
}
