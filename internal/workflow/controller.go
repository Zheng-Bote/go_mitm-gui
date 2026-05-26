// Package workflow provides a state machine for the load→validate→upload workflow.
package workflow

import (
	"fmt"
	"sync"

	"github.com/zheng-bote/go_mitm-gui/internal/logging"
	"github.com/zheng-bote/go_mitm-gui/internal/model"
)

// State represents the current workflow state.
type State int

const (
	StateIdle State = iota
	StateConfigLoaded
	StateDataLoaded
	StateValidating
	StateValidated
	StateValidatedOk
	StateValidatedFail
	StateUploading
	StateUploaded
	StateUploadedOk
	StateUploadedFail
	StateError
)

func (s State) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateConfigLoaded:
		return "Config Loaded"
	case StateDataLoaded:
		return "Data Loaded"
	case StateValidating:
		return "Validating"
	case StateValidated:
		return "Validated"
	case StateValidatedOk:
		return "Validated ✓"
	case StateValidatedFail:
		return "Validated ✗"
	case StateUploading:
		return "Uploading"
	case StateUploaded:
		return "Uploaded"
	case StateUploadedOk:
		return "Uploaded ✓"
	case StateUploadedFail:
		return "Uploaded ✗"
	case StateError:
		return "Error"
	}
	return "Unknown"
}

// StateChangeCallback is called on every state transition.
type StateChangeCallback func(from, to State)

// Controller orchestrates the workflow and enforces valid transitions.
type Controller struct {
	mu         sync.RWMutex
	state      State
	config     *model.AppConfig
	data       *model.LoadedData
	validation *model.ValidationResult
	upload     *model.UploadResult
	log        *logging.Logger
	onChange   StateChangeCallback
}

// NewController creates a new workflow controller.
func NewController(logger *logging.Logger) *Controller {
	return &Controller{
		state: StateIdle,
		log:   logger,
	}
}

// SetStateChangeCallback registers a callback for state transitions.
func (c *Controller) SetStateChangeCallback(cb StateChangeCallback) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onChange = cb
}

// State returns the current state.
func (c *Controller) State() State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// Config returns the current config.
func (c *Controller) Config() *model.AppConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// Data returns the currently loaded data.
func (c *Controller) Data() *model.LoadedData {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data
}

// ValidationResult returns the last validation result.
func (c *Controller) ValidationResult() *model.ValidationResult {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.validation
}

// UploadResult returns the last upload result.
func (c *Controller) UploadResult() *model.UploadResult {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.upload
}

// SetConfig sets the configuration and transitions to ConfigLoaded.
func (c *Controller) SetConfig(cfg *model.AppConfig) {
	c.mu.Lock()
	c.config = cfg
	c.mu.Unlock()
	c.transition(StateConfigLoaded)
	c.log.Info("Configuration loaded: %d topic(s), admins: %v", len(cfg.Topics), cfg.Global.Admins)
}

// SetData sets the loaded data and transitions to DataLoaded.
func (c *Controller) SetData(data *model.LoadedData) error {
	state := c.State()
	if state != StateConfigLoaded && state != StateValidatedFail && state != StateUploadedFail {
		return fmt.Errorf("workflow: cannot load data in state %s", state)
	}
	c.mu.Lock()
	c.data = data
	c.mu.Unlock()
	c.transition(StateDataLoaded)
	c.log.Info("Data loaded: %d rows from %s (fields: %v)", data.RowCount, data.SourcePath, data.Fields)
	return nil
}

// SetValidation sets the validation result.
func (c *Controller) SetValidation(result *model.ValidationResult) error {
	state := c.State()
	if state != StateDataLoaded {
		return fmt.Errorf("workflow: cannot validate in state %s", state)
	}
	c.mu.Lock()
	c.validation = result
	c.mu.Unlock()

	if result.Valid {
		c.transition(StateValidatedOk)
		c.log.Info("Validation passed: %d errors, %d warnings", len(result.Errors), len(result.Warnings))
	} else {
		c.transition(StateValidatedFail)
		c.log.Warn("Validation failed: %d errors, %d warnings", len(result.Errors), len(result.Warnings))
	}
	return nil
}

// SetUpload sets the upload result.
func (c *Controller) SetUpload(result *model.UploadResult) error {
	state := c.State()
	if state != StateValidatedOk && state != StateValidatedFail && state != StateDataLoaded {
		return fmt.Errorf("workflow: cannot upload in state %s", state)
	}
	c.mu.Lock()
	c.upload = result
	c.mu.Unlock()

	if result.Success {
		c.transition(StateUploadedOk)
		c.log.Info("Upload succeeded: HTTP %d", result.StatusCode)
	} else {
		c.transition(StateUploadedFail)
		c.log.Error("Upload failed: HTTP %d - %s", result.StatusCode, result.ResponseBody)
	}
	return nil
}

// CanLoad returns true if data can be loaded in the current state.
func (c *Controller) CanLoad() bool {
	s := c.State()
	return s == StateConfigLoaded || s == StateValidatedFail || s == StateUploadedFail
}

// CanValidate returns true if validation can be performed.
func (c *Controller) CanValidate() bool {
	return c.State() == StateDataLoaded
}

// CanUpload returns true if upload can be performed.
func (c *Controller) CanUpload() bool {
	return c.State() == StateValidatedOk
}

// Reset resets the controller to idle.
func (c *Controller) Reset() {
	c.mu.Lock()
	c.data = nil
	c.validation = nil
	c.upload = nil
	c.mu.Unlock()
	c.transition(StateIdle)
	c.log.Info("Workflow reset")
}

// ResetToConfig resets data but keeps config, transitions to ConfigLoaded.
func (c *Controller) ResetToConfig() {
	c.mu.Lock()
	c.data = nil
	c.validation = nil
	c.upload = nil
	c.mu.Unlock()
	c.transition(StateConfigLoaded)
	c.log.Info("Workflow reset to config loaded")
}

func (c *Controller) transition(newState State) {
	c.mu.Lock()
	oldState := c.state
	c.state = newState
	cb := c.onChange
	c.mu.Unlock()

	c.log.Debug("State: %s → %s", oldState, newState)
	if cb != nil {
		cb(oldState, newState)
	}
}
