// Package gui provides the Fyne-based graphical interface for mitm-gui.
package gui

import (
	"encoding/json"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/zheng-bote/go_mitm-gui/internal/config"
	"github.com/zheng-bote/go_mitm-gui/internal/crypto"
	"github.com/zheng-bote/go_mitm-gui/internal/logging"
	"github.com/zheng-bote/go_mitm-gui/internal/model"
)

type App struct {
	window      fyne.Window
	config      *model.AppConfig
	proxyConfig *model.ProxyConfig
	logger      *logging.Logger

	tabs     *container.AppTabs
	settings *SettingsTab
	workflow *WorkflowTab

	statusLabel *widget.Label
}

func NewApp(w fyne.Window, logPath string) *App {
	logger, err := logging.New(logPath)
	if err != nil {
		logger = logging.NopLogger()
	}

	a := &App{
		window:      w,
		logger:      logger,
		statusLabel: widget.NewLabel("Ready - load an encrypted INI file in Settings."),
	}

	a.settings = NewSettingsTab(w, a.onConfigLoaded, a.onProxySave)
	a.workflow = NewWorkflowTab(w, logger)

	a.loadProxyConfig()
	a.workflow.SetProxyConfig(a.proxyConfig)

	a.tabs = container.NewAppTabs(
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), a.settings.Build()),
		container.NewTabItemWithIcon("Workflow", theme.ComputerIcon(), a.workflow.Build()),
	)
	a.tabs.DisableIndex(1)

	content := container.NewBorder(
		nil,
		container.NewHBox(widget.NewLabel("Status:"), a.statusLabel),
		nil, nil,
		a.tabs,
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(800, 600))
	logger.Info("Application started")
	return a
}

func (a *App) loadProxyConfig() {
	username := os.Getenv("USERNAME")
	if username == "" {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	dir := filepath.Dir(exe)
	proxyPath := filepath.Join(dir, "proxy_"+username+".enc")
	if _, err := os.Stat(proxyPath); os.IsNotExist(err) {
		return
	}
	a.logger.Info("Proxy file found: %s (load via Save Proxy with password)", proxyPath)
}

func (a *App) onConfigLoaded(cfg *model.AppConfig) {
	a.config = cfg
	if cfg.Global.LogFilePath != "" {
		level := logging.ParseLevel(cfg.Global.LogLevel)
		if err := a.logger.Reconfigure(cfg.Global.LogFilePath, level); err != nil {
			a.logger.Warn("Could not reconfigure log file: %v", err)
		}
	}
	a.logger.Info("Configuration loaded from encrypted file")
	a.statusLabel.SetText("Configuration loaded successfully.")
	a.tabs.EnableIndex(1)
	a.tabs.SelectIndex(1)
	a.workflow.SetConfig(cfg)
}

func (a *App) onProxySave(proxy *model.ProxyConfig) {
	a.proxyConfig = proxy
	if a.workflow != nil {
		a.workflow.SetProxyConfig(proxy)
	}
	a.logger.Info("Proxy configuration saved")

	username := os.Getenv("USERNAME")
	if username == "" {
		username = "default"
	}
	if a.settings.iniPath != "" && proxy.Password != "" {
		proxyDir := proxyFileDir(a.settings.iniPath)
		encPath := proxyDir + "\\proxy_" + username + ".enc"

		proxyJSON, _ := json.Marshal(proxy)
		encrypted, err := crypto.EncryptFile("proxy-"+proxy.Password, proxyJSON)
		if err != nil {
			a.logger.Warn("Failed to encrypt proxy config: %v", err)
			return
		}
		os.WriteFile(encPath, encrypted, 0600)
		a.logger.Info("Proxy config saved to %s", encPath)
	}
}

func proxyFileDir(iniPath string) string {
	for i := len(iniPath) - 1; i >= 0; i-- {
		if iniPath[i] == '\\' || iniPath[i] == '/' {
			return iniPath[:i]
		}
	}
	return "."
}

func (a *App) SetStatus(text string) {
	a.statusLabel.SetText(text)
}

func (a *App) Config() *model.AppConfig { return a.config }
func (a *App) ProxyConfig() *model.ProxyConfig { return a.proxyConfig }
func (a *App) Loader() *config.Loader { return config.New() }
func (a *App) Logger() *logging.Logger { return a.logger }
