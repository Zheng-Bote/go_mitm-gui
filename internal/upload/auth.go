// Package upload provides HTTP upload with SaaS provider authentication.
//
// Auth flow (two-legged OAuth2-like):
//
//	1. POST /api/refreshtoken  →  refresh_token
//	2. GET  /api/token/        →  access_token (via Bearer refresh_token)
//	3. POST /api/employeeimport → upload (via Bearer access_token)
package upload

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/zheng-bote/go_mitm-gui/internal/model"
)

// AuthClient handles the refresh → access token flow.
type AuthClient struct {
	httpClient *http.Client
	config     *model.AuthConfig
}

// TokenSet holds both refresh and access tokens.
type TokenSet struct {
	RefreshToken string
	AccessToken  string
}

// NewAuthClient creates an AuthClient with the given auth config.
func NewAuthClient(config *model.AuthConfig, proxy *model.ProxyConfig) *AuthClient {
	var httpClient *http.Client
	if proxy != nil && proxy.Server != "" {
		proxyURL := &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
		}
		if proxy.User != "" {
			proxyURL.User = url.UserPassword(proxy.User, proxy.Password)
		}
		httpClient = &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	} else {
		httpClient = &http.Client{}
	}
	return &AuthClient{
		httpClient: httpClient,
		config:     config,
	}
}

// Authenticate performs the full auth flow: refresh → access token.
func (a *AuthClient) Authenticate() (*TokenSet, error) {
	// Step 1: Get refresh token.
	rtURL := strings.TrimRight(a.config.BaseURL, "/") + "/api/refreshtoken"

	loginPayload := map[string]interface{}{
		"user": map[string]string{
			"LoginName":     a.config.Login,
			"Loginpassword": a.config.Password,
		},
	}
	body, _ := json.Marshal(loginPayload)

	req1, err := http.NewRequest("POST", rtURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("auth: failed to create refresh request: %w", err)
	}
	req1.Header.Set("Content-Type", "application/json")

	resp1, err := a.httpClient.Do(req1)
	if err != nil {
		return nil, fmt.Errorf("auth: refresh token request failed: %w", err)
	}
	defer resp1.Body.Close()

	if resp1.StatusCode < 200 || resp1.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp1.Body)
		return nil, fmt.Errorf("auth: refresh token returned HTTP %d: %s", resp1.StatusCode, string(respBody))
	}

	var rtResult map[string]interface{}
	if err := json.NewDecoder(resp1.Body).Decode(&rtResult); err != nil {
		return nil, fmt.Errorf("auth: failed to parse refresh response: %w", err)
	}

	refreshToken, _ := rtResult["Token"].(string)
	if refreshToken == "" {
		refreshToken, _ = rtResult["token"].(string)
	}
	if refreshToken == "" {
		return nil, fmt.Errorf("auth: no token in refresh response: %v", rtResult)
	}

	// Step 2: Get access token.
	atURL := strings.TrimRight(a.config.BaseURL, "/") + "/api/token/"

	req2, err := http.NewRequest("GET", atURL, nil)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to create access token request: %w", err)
	}
	req2.Header.Set("Authorization", "Bearer "+refreshToken)

	resp2, err := a.httpClient.Do(req2)
	if err != nil {
		return nil, fmt.Errorf("auth: access token request failed: %w", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode < 200 || resp2.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp2.Body)
		return nil, fmt.Errorf("auth: access token returned HTTP %d: %s", resp2.StatusCode, string(respBody))
	}

	var atResult map[string]interface{}
	if err := json.NewDecoder(resp2.Body).Decode(&atResult); err != nil {
		return nil, fmt.Errorf("auth: failed to parse access token response: %w", err)
	}

	accessToken, _ := atResult["AccessToken"].(string)
	if accessToken == "" {
		accessToken, _ = atResult["access_token"].(string)
	}
	if accessToken == "" {
		accessToken, _ = atResult["token"].(string)
	}
	if accessToken == "" {
		return nil, fmt.Errorf("auth: no access token in response: %v", atResult)
	}

	return &TokenSet{
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}, nil
}

// AuthenticatedUpload performs a full auth-then-upload flow.
// Returns the UploadResult which includes the auth error if auth fails.
func AuthenticatedUpload(
	recordsJSON []byte,
	uploadURL string,
	authConfig *model.AuthConfig,
	opts *model.UploadOptions,
	proxy *model.ProxyConfig,
) *model.UploadResult {
	client := NewAuthClient(authConfig, proxy)

	tokens, err := client.Authenticate()
	if err != nil {
		return &model.UploadResult{
			Success: false,
			Error:   fmt.Sprintf("Authentication failed: %v", err),
		}
	}

	// Build the envelope.
	var records []interface{}
	if err := json.Unmarshal(recordsJSON, &records); err != nil {
		return &model.UploadResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse records: %v", err),
		}
	}

	options := model.DefaultUploadOptions()
	if opts != nil {
		options = *opts
	}

	envelope := model.UploadEnvelope{Options: options, Records: records}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return &model.UploadResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to marshal envelope: %v", err),
		}
	}

	// Upload with Bearer token.
	return doUpload(client.httpClient, uploadURL, payload, tokens.AccessToken)
}

// doUpload sends the payload with Bearer auth.
func doUpload(httpClient *http.Client, uploadURL string, payload []byte, accessToken string) *model.UploadResult {
	req, err := http.NewRequest("POST", uploadURL, strings.NewReader(string(payload)))
	if err != nil {
		return &model.UploadResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to create upload request: %v", err),
		}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return &model.UploadResult{
			Success: false,
			Error:   fmt.Sprintf("HTTP request failed: %v", err),
		}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	return &model.UploadResult{
		Success:      success,
		StatusCode:   resp.StatusCode,
		ResponseBody: string(body),
	}
}
