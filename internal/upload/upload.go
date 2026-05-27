// Package upload handles HTTP POST upload of validated data with envelope metadata.
package upload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/zheng-bote/go_mitm-gui/internal/model"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{}}
}

func NewClientWithProxy(proxy *model.ProxyConfig) *Client {
	transport := &http.Transport{}
	if proxy != nil && proxy.Server != "" {
		proxyURL := &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
		}
		if proxy.User != "" {
			proxyURL.User = url.UserPassword(proxy.User, proxy.Password)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return &Client{httpClient: &http.Client{Transport: transport}}
}

func (c *Client) Upload(recordsJSON []byte, uploadURL string, opts *model.UploadOptions) (*model.UploadResult, error) {
	if uploadURL == "" {
		return nil, fmt.Errorf("upload: upload URL is empty")
	}

	options := model.DefaultUploadOptions()
	if opts != nil {
		options = *opts
	}

	envelope := model.UploadEnvelope{Options: options, Records: json.RawMessage(recordsJSON)}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("upload: failed to marshal envelope: %w", err)
	}

	req, err := http.NewRequest("POST", uploadURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("upload: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &model.UploadResult{
			Success: false, Error: fmt.Sprintf("HTTP request failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &model.UploadResult{
			Success: false, StatusCode: resp.StatusCode,
			Error: fmt.Sprintf("Failed to read response body: %v", err),
		}, nil
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	return &model.UploadResult{
		Success:      success,
		StatusCode:   resp.StatusCode,
		ResponseBody: string(body),
	}, nil
}

func BuildEnvelopeJSON(recordsJSON []byte, opts *model.UploadOptions) ([]byte, error) {
	options := model.DefaultUploadOptions()
	if opts != nil {
		options = *opts
	}
	return json.Marshal(model.UploadEnvelope{Options: options, Records: json.RawMessage(recordsJSON)})
}
