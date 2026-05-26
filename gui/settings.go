package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/zheng-bote/go_mitm-gui/internal/config"
	"github.com/zheng-bote/go_mitm-gui/internal/model"
)

type onConfigLoaded func(*model.AppConfig)

type SettingsTab struct {
	parent     fyne.Window
	onLoaded   onConfigLoaded
	onProxySave func(*model.ProxyConfig)

	iniPathLabel *widget.Label
	iniPath      string

	masterPassword *widget.Entry
	proxyPassword  *widget.Entry

	proxyServer *widget.Entry
	proxyPort   *widget.Entry
	proxyUser   *widget.Entry
	proxyPass   *widget.Entry

	loadBtn    *widget.Button
	selectBtn  *widget.Button
	saveProxyBtn *widget.Button
}

func NewSettingsTab(parent fyne.Window, onLoaded onConfigLoaded, onProxySave func(*model.ProxyConfig)) *SettingsTab {
	st := &SettingsTab{
		parent:      parent,
		onLoaded:    onLoaded,
		onProxySave: onProxySave,
	}

	st.iniPathLabel = widget.NewLabel("(none selected)")

	st.masterPassword = widget.NewPasswordEntry()
	st.masterPassword.PlaceHolder = "Enter master password"

	st.proxyPassword = widget.NewPasswordEntry()
	st.proxyPassword.PlaceHolder = "Enter proxy config password"

	st.proxyServer = widget.NewEntry()
	st.proxyServer.PlaceHolder = "e.g. proxy.example.com"

	st.proxyPort = widget.NewEntry()
	st.proxyPort.PlaceHolder = "e.g. 3128"

	st.proxyUser = widget.NewEntry()
	st.proxyUser.PlaceHolder = "Proxy username (optional)"

	st.proxyPass = widget.NewPasswordEntry()
	st.proxyPass.PlaceHolder = "Proxy password (optional)"

	st.selectBtn = widget.NewButton("Select Encrypted INI File...", st.onSelectFile)
	st.loadBtn = widget.NewButton("Load Configuration", st.onLoadConfig)
	st.loadBtn.Disable()

	st.saveProxyBtn = widget.NewButton("Save Proxy Configuration", st.onSaveProxy)
	st.saveProxyBtn.Disable()

	return st
}

func (st *SettingsTab) Build() fyne.CanvasObject {
	iniSection := container.NewVBox(
		widget.NewLabelWithStyle("Encrypted INI Configuration", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		st.selectBtn,
		container.NewHBox(
			widget.NewLabel("Selected file:"),
			st.iniPathLabel,
		),
		widget.NewSeparator(),
		widget.NewLabel("Master Password:"),
		st.masterPassword,
	)

	proxySection := container.NewVBox(
		widget.NewLabelWithStyle("Proxy Configuration (optional)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Server:"),
		st.proxyServer,
		widget.NewLabel("Port:"),
		st.proxyPort,
		widget.NewLabel("User:"),
		st.proxyUser,
		widget.NewLabel("Password:"),
		st.proxyPass,
		widget.NewSeparator(),
		widget.NewLabel("Proxy Config File Password:"),
		st.proxyPassword,
	)

	return container.NewVBox(
		iniSection,
		widget.NewSeparator(),
		proxySection,
		widget.NewSeparator(),
		st.loadBtn,
		widget.NewSeparator(),
		st.saveProxyBtn,
	)
}

func (st *SettingsTab) onSelectFile() {
	dlg := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(fmt.Errorf("Error opening file: %w", err), st.parent)
			return
		}
		if reader == nil {
			return
		}
		st.iniPath = reader.URI().Path()
		st.iniPathLabel.SetText(st.iniPath)
		st.loadBtn.Enable()
		st.saveProxyBtn.Enable()
		_ = reader.Close()
	}, st.parent)

	// Try to set initial directory to data/ folder.
	dlg.SetFilter(storage.NewExtensionFileFilter([]string{".enc"}))
	dlg.Show()
}

func (st *SettingsTab) onLoadConfig() {
	if st.iniPath == "" {
		dialog.ShowInformation("No File", "Please select an encrypted INI file first.", st.parent)
		return
	}

	masterPwd := st.masterPassword.Text
	if masterPwd == "" {
		dialog.ShowInformation("Password Required", "Please enter the master password.", st.parent)
		return
	}

	loader := config.New()
	cfg, err := loader.LoadConfig(st.iniPath, masterPwd)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to load config: %w", err), st.parent)
		return
	}

	if st.onLoaded != nil {
		st.onLoaded(cfg)
	}

	dialog.ShowInformation("Success",
		fmt.Sprintf("Configuration loaded.\nTopics: %d\nAdmins: %v",
			len(cfg.Topics), cfg.Global.Admins), st.parent)
}


func (st *SettingsTab) onSaveProxy() {
	proxy := &model.ProxyConfig{
		Server:   st.proxyServer.Text,
		Port:     0,
		User:     st.proxyUser.Text,
		Password: st.proxyPass.Text,
	}
	fmt.Sscanf(st.proxyPort.Text, "%d", &proxy.Port)

	st.onProxySave(proxy)
}
