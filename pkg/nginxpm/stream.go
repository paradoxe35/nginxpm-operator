package nginxpm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Stream struct {
	ID             int       `json:"id"`
	UnscopedConfig *string   `json:"unscoped_config"` // Custom field from https://github.com/paradoxe35/nginx-proxy-manager, it must be a string pointer
	Meta           NginxMeta `json:"meta"`
}

type StreamRequestInput struct {
	IncomingPort   int    `json:"incoming_port"`
	ForwardingHost string `json:"forwarding_host"`
	ForwardingPort int    `json:"forwarding_port"`
	TCPForwarding  bool   `json:"tcp_forwarding"`
	CertificateID  int    `json:"certificate_id"`
	UDPForwarding  bool   `json:"udp_forwarding"`
	CustomFields   RequestCustomFields
}

// DeleteStream deletes a stream by its ID.
func (c *Client) DeleteStream(id int) error {
	endpoint := fmt.Sprintf("/api/nginx/streams/%d", id)
	resp, err := c.doRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("delete stream %d: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete stream %d: unexpected status code: %d", id, resp.StatusCode)
	}

	return nil
}

// FindStreamByID searches for an existing stream by its ID.
func (c *Client) FindStreamByID(id int) (*Stream, error) {
	resp, err := c.doRequest(http.MethodGet, "/api/nginx/streams", nil)
	if err != nil {
		return nil, fmt.Errorf("get streams: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get streams: unexpected status code: %d", resp.StatusCode)
	}

	var streams []Stream
	if err := json.NewDecoder(resp.Body).Decode(&streams); err != nil {
		return nil, fmt.Errorf("get streams: decode response: %w", err)
	}

	for _, stream := range streams {
		if stream.ID == id {
			return &stream, nil
		}
	}

	return nil, nil // No matching stream found
}

// CreateStream creates a new stream.
func (c *Client) CreateStream(input StreamRequestInput) (*Stream, error) {
	jsonBody, err := json.Marshal(buildStreamRequestBody(input))
	if err != nil {
		return nil, fmt.Errorf("create stream: marshal request: %w", err)
	}

	resp, err := c.doRequest(http.MethodPost, "/api/nginx/streams", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create stream: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create stream: unexpected status code: %d, body: %s",
			resp.StatusCode, string(respBody))
	}

	var newStream Stream
	if err := json.NewDecoder(resp.Body).Decode(&newStream); err != nil {
		return nil, fmt.Errorf("create stream: decode response: %w", err)
	}

	return &newStream, nil
}

// UpdateStream updates an existing stream.
func (c *Client) UpdateStream(id int, input StreamRequestInput) (*Stream, error) {
	jsonBody, err := json.Marshal(buildStreamRequestBody(input))
	if err != nil {
		return nil, fmt.Errorf("update stream %d: marshal request: %w", id, err)
	}

	endpoint := fmt.Sprintf("/api/nginx/streams/%d", id)
	resp, err := c.doRequest(http.MethodPut, endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("update stream %d: request failed: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("update stream %d: unexpected status code: %d, body: %s",
			id, resp.StatusCode, string(respBody))
	}

	var updatedStream Stream
	if err := json.NewDecoder(resp.Body).Decode(&updatedStream); err != nil {
		return nil, fmt.Errorf("update stream %d: decode response: %w", id, err)
	}

	return &updatedStream, nil
}

// buildStreamRequestBody creates the request body for stream operations.
func buildStreamRequestBody(input StreamRequestInput) map[string]interface{} {
	body := map[string]interface{}{
		"incoming_port":   input.IncomingPort,
		"forwarding_host": input.ForwardingHost,
		"forwarding_port": input.ForwardingPort,
		"tcp_forwarding":  input.TCPForwarding,
		"udp_forwarding":  input.UDPForwarding,
		"certificate_id":  input.CertificateID,
		"meta": map[string]interface{}{
			"letsencrypt_agree":        false,
			"dns_challenge":            true,
			"dns_provider_credentials": "",
		},
	}

	if input.CustomFields != nil {
		for _, custom := range input.CustomFields {
			if custom.Allowed {
				body[custom.Field] = custom.Value
			}
		}
	}

	return body
}
