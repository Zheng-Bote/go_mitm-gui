package upload

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zheng-bote/go_mitm-gui/internal/model"
)

const sampleRecords = `[
	{"EmployeeNumber":"100001","FirstName":"John","LastName":"Doe"},
	{"EmployeeNumber":"100002","FirstName":"Jane","LastName":"Smith"}
]`

// --- Envelope tests ---

func TestBuildEnvelopeJSON(t *testing.T) {
	p, err := BuildEnvelopeJSON([]byte(sampleRecords), nil)
	if err != nil {
		t.Fatal(err)
	}
	var env model.UploadEnvelope
	json.Unmarshal(p, &env)
	if env.Options.UpdateExistingRecords != "true" {
		t.Fatalf("expected true, got %q", env.Options.UpdateExistingRecords)
	}
	
	var records []interface{}
	if err := json.Unmarshal(env.Records, &records); err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
}

func TestBuildEnvelopeJSON_CustomOptions(t *testing.T) {
	opts := model.UploadOptions{UpdateExistingRecords: "false", DateFormat: "yyyy-mm-dd"}
	p, err := BuildEnvelopeJSON([]byte(sampleRecords), &opts)
	if err != nil {
		t.Fatal(err)
	}
	var env model.UploadEnvelope
	json.Unmarshal(p, &env)
	if env.Options.UpdateExistingRecords != "false" {
		t.Fatalf("expected false, got %q", env.Options.UpdateExistingRecords)
	}
}

// --- Direct upload tests (no auth) ---

func TestUpload_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	client := NewClient()
	result, err := client.Upload([]byte(sampleRecords), srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success, got %d: %s", result.StatusCode, result.ResponseBody)
	}
}

func TestUpload_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid"}`))
	}))
	defer srv.Close()

	client := NewClient()
	result, err := client.Upload([]byte(sampleRecords), srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Fatal("expected failure")
	}
	if result.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", result.StatusCode)
	}
}

func TestUpload_EmptyURL(t *testing.T) {
	_, err := NewClient().Upload([]byte(`[]`), "", nil)
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestNewClientWithProxy(t *testing.T) {
	proxy := &model.ProxyConfig{Server: "proxy.example.com", Port: 3128}
	client := NewClientWithProxy(proxy)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

// --- Auth flow tests ---

// authTestServer simulates the two-legged auth + upload flow.
func authTestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/refreshtoken":
			// Return refresh token.
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"Token": "rt_mock_refresh_token_123",
			})

		case r.Method == "GET" && r.URL.Path == "/api/token/":
			// Verify Bearer token, return access token.
			if r.Header.Get("Authorization") != "Bearer rt_mock_refresh_token_123" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"AccessToken": "at_mock_access_token_456",
			})

		case r.Method == "POST" && r.URL.Path == "/api/employeeimport":
			// Verify Bearer token on upload.
			if r.Header.Get("Authorization") != "Bearer at_mock_access_token_456" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":   "ok",
				"uploaded": 2,
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestAuthenticatedUpload_Success(t *testing.T) {
	srv := authTestServer(t)
	defer srv.Close()

	authConfig := &model.AuthConfig{
		BaseURL:  srv.URL,
		Login:    "testuser",
		Password: "testpass",
	}

	result := AuthenticatedUpload(
		[]byte(sampleRecords),
		srv.URL+"/api/employeeimport",
		authConfig,
		nil,
		nil,
	)

	if !result.Success {
		t.Fatalf("expected success, got HTTP %d: %s (error: %s)",
			result.StatusCode, result.ResponseBody, result.Error)
	}
	if result.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", result.StatusCode)
	}
}

func TestAuthClient_Authenticate(t *testing.T) {
	srv := authTestServer(t)
	defer srv.Close()

	client := NewAuthClient(&model.AuthConfig{
		BaseURL:  srv.URL,
		Login:    "testuser",
		Password: "testpass",
	}, nil)

	tokens, err := client.Authenticate()
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	if tokens.RefreshToken != "rt_mock_refresh_token_123" {
		t.Fatalf("unexpected refresh token: %q", tokens.RefreshToken)
	}
	if tokens.AccessToken != "at_mock_access_token_456" {
		t.Fatalf("unexpected access token: %q", tokens.AccessToken)
	}
}

func TestAuthClient_InvalidCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := NewAuthClient(&model.AuthConfig{
		BaseURL:  srv.URL,
		Login:    "baduser",
		Password: "badpass",
	}, nil)

	_, err := client.Authenticate()
	if err == nil {
		t.Fatal("expected error for invalid credentials")
	}
}
