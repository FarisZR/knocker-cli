package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

type KnockResponse struct {
	WhitelistedEntry string `json:"whitelisted_entry"`
	ExpiresAt        int64  `json:"expires_at"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

func NewClient(baseURL string, apiKey string) *Client {
	return &Client{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) HealthCheck() error {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/health", c.BaseURL), nil)
	if err != nil {
		return err
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status code: %d", res.StatusCode)
	}

	return nil
}

func (c *Client) Knock(ipAddress string, ttl int) (*KnockResponse, error) {
	requestBody := map[string]interface{}{}
	if ipAddress != "" {
		requestBody["ip_address"] = ipAddress
	}
	if ttl > 0 {
		requestBody["ttl"] = ttl
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/knock", c.BaseURL), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.APIKey)

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("knock failed with status code: %d", res.StatusCode)
	}

	var knockResponse KnockResponse
	if err := json.NewDecoder(res.Body).Decode(&knockResponse); err != nil {
		return nil, err
	}

	return &knockResponse, nil
}