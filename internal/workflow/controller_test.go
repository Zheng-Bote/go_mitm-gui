package workflow

import (
	"testing"

	"github.com/zheng-bote/go_mitm-gui/internal/logging"
	"github.com/zheng-bote/go_mitm-gui/internal/model"
)

func TestController_InitialState(t *testing.T) {
	c := NewController(logging.NopLogger())
	if c.State() != StateIdle {
		t.Fatalf("expected StateIdle, got %s", c.State())
	}
}

func TestController_SetConfig(t *testing.T) {
	c := NewController(logging.NopLogger())
	cfg := &model.AppConfig{Topics: map[string]*model.TopicConfig{"HR": {}}}
	c.SetConfig(cfg)
	if c.State() != StateConfigLoaded {
		t.Fatalf("expected ConfigLoaded, got %s", c.State())
	}
	if c.Config() != cfg {
		t.Fatal("config not stored")
	}
}

func TestController_LoadThenValidate(t *testing.T) {
	c := NewController(logging.NopLogger())
	c.SetConfig(&model.AppConfig{})

	err := c.SetData(&model.LoadedData{RowCount: 5})
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	if c.State() != StateDataLoaded {
		t.Fatalf("expected DataLoaded, got %s", c.State())
	}

	err = c.SetValidation(&model.ValidationResult{Valid: true})
	if err != nil {
		t.Fatalf("SetValidation failed: %v", err)
	}
	if c.State() != StateValidatedOk {
		t.Fatalf("expected ValidatedOk, got %s", c.State())
	}
}

func TestController_InvalidTransition(t *testing.T) {
	c := NewController(logging.NopLogger())

	// Can't validate without data.
	err := c.SetValidation(&model.ValidationResult{Valid: true})
	if err == nil {
		t.Fatal("expected error for invalid transition")
	}
}

func TestController_CanChecks(t *testing.T) {
	c := NewController(logging.NopLogger())

	if c.CanLoad() {
		t.Fatal("CanLoad should be false in Idle")
	}
	if c.CanValidate() {
		t.Fatal("CanValidate should be false in Idle")
	}
	if c.CanUpload() {
		t.Fatal("CanUpload should be false in Idle")
	}

	c.SetConfig(&model.AppConfig{})

	if !c.CanLoad() {
		t.Fatal("CanLoad should be true after ConfigLoaded")
	}
	if c.CanValidate() {
		t.Fatal("CanValidate should be false after ConfigLoaded")
	}

	c.SetData(&model.LoadedData{RowCount: 1})

	if !c.CanValidate() {
		t.Fatal("CanValidate should be true after DataLoaded")
	}
}

func TestController_StateCallback(t *testing.T) {
	c := NewController(logging.NopLogger())
	var transitions []string
	c.SetStateChangeCallback(func(from, to State) {
		transitions = append(transitions, from.String()+"→"+to.String())
	})

	c.SetConfig(&model.AppConfig{})
	c.SetData(&model.LoadedData{RowCount: 1})

	if len(transitions) != 2 {
		t.Fatalf("expected 2 transitions, got %d: %v", len(transitions), transitions)
	}
}

func TestController_Reset(t *testing.T) {
	c := NewController(logging.NopLogger())
	c.SetConfig(&model.AppConfig{})
	c.SetData(&model.LoadedData{RowCount: 1})

	c.Reset()
	if c.State() != StateIdle {
		t.Fatalf("expected Idle after reset, got %s", c.State())
	}
	if c.Data() != nil {
		t.Fatal("data should be nil after reset")
	}
}
