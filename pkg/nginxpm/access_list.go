package nginxpm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type AccessList struct {
	ID             int `json:"id"`
	ProxyHostCount int `json:"proxy_host_count"`
}

type AccessListItem struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AccessListClient struct {
	Address   string `json:"address"`
	Directive string `json:"directive"`
}

type AccessListRequestInput struct {
	Name       string             `json:"name"`
	SatisfyAny bool               `json:"satisfy_any"`
	PassAuth   bool               `json:"pass_auth"`
	Items      []AccessListItem   `json:"items"`
	Clients    []AccessListClient `json:"clients"`
}

// DeleteAccessList deletes a access list by its ID.
func (c *Client) DeleteAccessList(id int) error {
	endpoint := fmt.Sprintf("/api/nginx/access-lists/%d", id)
	resp, err := c.doRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("delete access list %d: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete access list %d: unexpected status code: %d", id, resp.StatusCode)
	}

	return nil
}

// FindAccessListByID searches for an existing access list by its ID.
func (c *Client) FindAccessListByID(id int) (*AccessList, error) {
	resp, err := c.doRequest(http.MethodGet, "/api/nginx/access-lists", nil)
	if err != nil {
		return nil, fmt.Errorf("get access lists: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get access lists: unexpected status code: %d", resp.StatusCode)
	}

	var accesses []AccessList
	if err := json.NewDecoder(resp.Body).Decode(&accesses); err != nil {
		return nil, fmt.Errorf("get access lists: decode response: %w", err)
	}

	for _, access := range accesses {
		if access.ID == id {
			return &access, nil
		}
	}

	return nil, nil // No matching access list found
}

// CreateAccessList creates a new access list.
func (c *Client) CreateAccessList(input AccessListRequestInput) (*AccessList, error) {
	jsonBody, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("create access list: marshal request: %w", err)
	}

	resp, err := c.doRequest(http.MethodPost, "/api/nginx/access-lists", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create access list: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create access list: unexpected status code: %d, body: %s",
			resp.StatusCode, string(respBody))
	}

	var newAccessList AccessList
	if err := json.NewDecoder(resp.Body).Decode(&newAccessList); err != nil {
		return nil, fmt.Errorf("create access list: decode response: %w", err)
	}

	return &newAccessList, nil
}

// UpdateAccessList updates an existing access list.
func (c *Client) UpdateAccessList(id int, input AccessListRequestInput) (*AccessList, error) {
	jsonBody, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("update access list %d: marshal request: %w", id, err)
	}

	endpoint := fmt.Sprintf("/api/nginx/access-lists/%d", id)
	resp, err := c.doRequest(http.MethodPut, endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("update access list %d: request failed: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("update access list %d: unexpected status code: %d, body: %s",
			id, resp.StatusCode, string(respBody))
	}

	var updatedAccessList AccessList
	if err := json.NewDecoder(resp.Body).Decode(&updatedAccessList); err != nil {
		return nil, fmt.Errorf("update access list %d: decode response: %w", id, err)
	}

	return &updatedAccessList, nil
}
