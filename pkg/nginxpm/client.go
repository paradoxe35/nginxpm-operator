/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nginxpm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	nginxpmoperatoriov1 "github.com/paradoxe35/nginxpm-operator/api/v1"
)

const (
	NGINX_LB_SERVER_PREFIX = "xlb"
)

// APIError represents an error returned by the API
type APIError struct {
	StatusCode int
	Body       string
}

// Client represents the NGINX Proxy Manager API client.
// It contains the HTTP client, API endpoint, and authentication token.
type Client struct {
	httpClient *http.Client
	Endpoint   string
	Token      string
	Expires    time.Time
}

// TokenResponse represents the structure of the token response from the API.
// It contains the actual token and its expiration time.
type TokenResponse struct {
	Token   string    `json:"token"`
	Expires time.Time `json:"expires"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error: status code %d, body: %s", e.StatusCode, e.Body)
}

// NewClient creates a new instance of the NGINX Proxy Manager client.
// It takes the API endpoint as a parameter and sets up a default HTTP client with a timeout.
func NewClient(httpClient *http.Client, endpoint string) *Client {
	return &Client{
		Endpoint:   endpoint,
		httpClient: httpClient,
	}
}

// NewClientFromToken creates a new client instance with a pre-existing token.
// This is useful when you already have a valid token and don't need to authenticate.
func NewClientFromToken(httpClient *http.Client, token *nginxpmoperatoriov1.Token) *Client {
	var tokenValue string
	if token.Status.Token != nil {
		tokenValue = *token.Status.Token
	}

	var expiresValue time.Time
	if token.Status.Expires != nil {
		expiresValue = token.Status.Expires.Time
	}

	return &Client{
		Endpoint:   token.Spec.Endpoint,
		Token:      tokenValue,
		Expires:    expiresValue,
		httpClient: httpClient,
	}
}

// CreateClientToken creates a new token for the client.
// It takes the identity and secret as parameters and sends a POST request to the /api/tokens endpoint.
// It returns an error if the request fails or if the response status code is not 200.
func CreateClientToken(client *Client, identity string, secret string) error {
	payload := map[string]string{
		"identity": identity,
		"secret":   secret,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("[/api/tokens] Error marshaling payload: %w", err)
	}

	resp, err := client.doRequest(http.MethodPost, "/api/tokens", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("[/api/tokens] error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[/api/tokens] unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResponse TokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return fmt.Errorf("[/api/tokens] error unmarshaling response: %w", err)
	}

	// Store the token in the client for future requests
	client.Token = tokenResponse.Token
	client.Expires = tokenResponse.Expires

	return nil
}

// doRequest is a helper method that performs HTTP requests to the API.
// It sets up common headers, handles authentication, and performs the actual HTTP request.
func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	url := c.Endpoint + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	return resp, nil
}

// CheckConnection sends a GET request to the /api endpoint to verify connectivity.
// It returns nil if the connection is successful, or an error if it fails.
func (c *Client) CheckConnection() error {
	resp, err := c.doRequest(http.MethodGet, "/api/", nil)

	if err != nil {
		return fmt.Errorf("[/api/] Error checking connection: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[/api/] Unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// Check token user is valid
// It returns an error if it fails.
func (c *Client) CheckTokenAccess() error {
	resp, err := c.doRequest(http.MethodGet, "/api/users/me", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[/api/users/me] unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
