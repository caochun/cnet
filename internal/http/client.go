package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents an HTTP client with common functionality
type Client struct {
	httpClient *http.Client
	timeout    time.Duration
}

// NewClient creates a new HTTP client
func NewClient(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// Get performs a GET request
func (c *Client) Get(url string) (*http.Response, error) {
	return c.httpClient.Get(url)
}

// Post performs a POST request with JSON body
func (c *Client) Post(url string, data interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	return c.httpClient.Do(req)
}

// PostWithRetry performs a POST request with retry logic
func (c *Client) PostWithRetry(url string, data interface{}, maxRetries int) (*http.Response, error) {
	var lastErr error
	
	for i := 0; i < maxRetries; i++ {
		resp, err := c.Post(url, data)
		if err == nil && resp.StatusCode == http.StatusOK {
			return resp, nil
		}
		
		lastErr = err
		if resp != nil {
			resp.Body.Close()
		}
		
		// Wait before retry (exponential backoff)
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}
	
	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// GetJSON performs a GET request and unmarshals the response
func (c *Client) GetJSON(url string, target interface{}) error {
	resp, err := c.Get(url)
	if err != nil {
		return fmt.Errorf("failed to get %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// PostJSON performs a POST request with JSON body and unmarshals the response
func (c *Client) PostJSON(url string, data interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
