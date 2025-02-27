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
	"net/url"
)

const (
	CUSTOM_FIELD_UNSCOPED_CONFIG = "unscoped_config"
)

// ProxyHost represents the structure of a proxy host as returned by the API.
type ProxyHost struct {
	ID             int      `json:"id"`
	Enabled        bool     `json:"enabled"`
	UnscopedConfig *string  `json:"unscoped_config"` // Custom field from https://github.com/paradoxe35/nginx-proxy-manager, it must be a string pointer
	DomainNames    []string `json:"domain_names"`
}

// ProxyHostMeta represents the meta information for a proxy host.
type ProxyHostMeta struct {
	LetsEncryptAgree bool    `json:"letsencrypt_agree"`
	DNSChallenge     bool    `json:"dns_challenge"`
	NginxOnline      bool    `json:"nginx_online"`
	NginxErr         *string `json:"nginx_err"`
}

// ProxyHostLocation represents a location configuration for a proxy host.
type ProxyHostLocation struct {
	Path           string `json:"path"`
	AdvancedConfig string `json:"advanced_config"`
	ForwardScheme  string `json:"forward_scheme"`
	ForwardHost    string `json:"forward_host"`
	ForwardPort    int    `json:"forward_port"`
}

type RequestCustomField struct {
	Field   string
	Value   string
	Allowed bool
}

type ProxyHostRequestCustomFields map[string]RequestCustomField

// ProxyHostRequestInput holds all parameters needed to create a proxy host.
type ProxyHostRequestInput struct {
	DomainNames           []string
	ForwardHost           string
	ForwardScheme         string
	ForwardPort           int
	BlockExploits         bool
	AllowWebsocketUpgrade bool
	CertificateID         *int
	SSLForced             bool
	HTTP2Support          bool
	HSTSEnabled           bool
	AdvancedConfig        string
	Locations             []ProxyHostLocation
	CachingEnabled        bool
	HSTSSubdomains        bool
	AccessListID          int
	CustomFields          ProxyHostRequestCustomFields
}

// DeleteProxyHost deletes a proxy host by its ID.
func (c *Client) DeleteProxyHost(id int) error {
	endpoint := fmt.Sprintf("/api/nginx/proxy-hosts/%d", id)
	resp, err := c.doRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("delete proxy host %d: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete proxy host %d: unexpected status code: %d", id, resp.StatusCode)
	}

	return nil
}

// DisableProxyHost disable a proxy host by its ID.
func (c *Client) DisableProxyHost(id int) error {
	endpoint := fmt.Sprintf("/api/nginx/proxy-hosts/%d/disable", id)
	resp, err := c.doRequest(http.MethodPost, endpoint, nil)
	if err != nil {
		return fmt.Errorf("disable proxy host %d: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("disable proxy host %d: unexpected status code: %d", id, resp.StatusCode)
	}

	return nil
}

// EnableProxyHost enable a proxy host by its ID.
func (c *Client) EnableProxyHost(id int) error {
	endpoint := fmt.Sprintf("/api/nginx/proxy-hosts/%d/enable", id)
	resp, err := c.doRequest(http.MethodPost, endpoint, nil)
	if err != nil {
		return fmt.Errorf("enable proxy host %d: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("enable proxy host %d: unexpected status code: %d", id, resp.StatusCode)
	}

	return nil
}

// FindProxyHostByDomain searches for an existing proxy host matching the given domains.
func (c *Client) FindProxyHostByDomain(domains []string) (*ProxyHost, error) {
	if len(domains) == 0 {
		return nil, fmt.Errorf("find proxy host by domain: no domains provided")
	}

	query := url.QueryEscape(domains[0])
	endpoint := fmt.Sprintf("/api/nginx/proxy-hosts?query=%s", query)

	hosts, err := c.getProxyHosts(endpoint)
	if err != nil {
		return nil, fmt.Errorf("find proxy host by domain: %w", err)
	}

	// Check if any host has the exact domains we're looking for
	for _, host := range hosts {
		if !domainsMatch(host.DomainNames, domains) {
			continue
		}
		return &host, nil
	}

	return nil, nil // No matching proxy host found
}

// FindProxyHostByID searches for an existing proxy host by its ID.
func (c *Client) FindProxyHostByID(id int) (*ProxyHost, error) {
	hosts, err := c.getProxyHosts("/api/nginx/proxy-hosts")
	if err != nil {
		return nil, fmt.Errorf("find proxy host by ID: %w", err)
	}

	for _, host := range hosts {
		if host.ID == id {
			return &host, nil
		}
	}

	return nil, nil // No matching proxy host found
}

// CreateProxyHost creates a new proxy host.
func (c *Client) CreateProxyHost(input ProxyHostRequestInput) (*ProxyHost, error) {
	body := buildProxyHostRequestBody(input)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("create proxy host: marshal request: %w", err)
	}

	resp, err := c.doRequest(http.MethodPost, "/api/nginx/proxy-hosts", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create proxy host: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create proxy host: unexpected status code: %d, body: %s",
			resp.StatusCode, string(respBody))
	}

	var newProxyHost ProxyHost
	if err := json.NewDecoder(resp.Body).Decode(&newProxyHost); err != nil {
		return nil, fmt.Errorf("create proxy host: decode response: %w", err)
	}

	return &newProxyHost, nil
}

// UpdateProxyHost updates an existing proxy host.
func (c *Client) UpdateProxyHost(id int, input ProxyHostRequestInput) (*ProxyHost, error) {
	body := buildProxyHostRequestBody(input)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("update proxy host %d: marshal request: %w", id, err)
	}

	endpoint := fmt.Sprintf("/api/nginx/proxy-hosts/%d", id)
	resp, err := c.doRequest(http.MethodPut, endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("update proxy host %d: request failed: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("update proxy host %d: unexpected status code: %d, body: %s",
			id, resp.StatusCode, string(respBody))
	}

	var updatedProxyHost ProxyHost
	if err := json.NewDecoder(resp.Body).Decode(&updatedProxyHost); err != nil {
		return nil, fmt.Errorf("update proxy host %d: decode response: %w", id, err)
	}

	return &updatedProxyHost, nil
}

// Helper functions

// getProxyHosts performs a GET request to fetch proxy hosts.
func (c *Client) getProxyHosts(endpoint string) ([]ProxyHost, error) {
	resp, err := c.doRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("get proxy hosts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get proxy hosts: unexpected status code: %d", resp.StatusCode)
	}

	var hosts []ProxyHost
	if err := json.NewDecoder(resp.Body).Decode(&hosts); err != nil {
		return nil, fmt.Errorf("get proxy hosts: decode response: %w", err)
	}

	return hosts, nil
}

// domainsMatch checks if two slices of domains contain the same elements.
func domainsMatch(hostDomains, searchDomains []string) bool {
	if len(hostDomains) != len(searchDomains) {
		return false
	}

	// Check that every search domain is in the host domains
	for _, searchDomain := range searchDomains {
		found := false
		for _, hostDomain := range hostDomains {
			if hostDomain == searchDomain {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// buildProxyHostRequestBody creates the request body for proxy host operations.
func buildProxyHostRequestBody(input ProxyHostRequestInput) map[string]interface{} {
	certificateID := 0
	if input.CertificateID != nil {
		certificateID = *input.CertificateID
	}

	body := map[string]interface{}{
		"domain_names":            input.DomainNames,
		"forward_host":            input.ForwardHost,
		"forward_scheme":          input.ForwardScheme,
		"forward_port":            input.ForwardPort,
		"block_exploits":          input.BlockExploits,
		"allow_websocket_upgrade": input.AllowWebsocketUpgrade,
		"access_list_id":          "0",
		"certificate_id":          certificateID,
		"ssl_forced":              input.SSLForced,
		"http2_support":           input.HTTP2Support,
		"hsts_enabled":            input.HSTSEnabled,
		"meta": map[string]interface{}{
			"letsencrypt_agree": false,
			"dns_challenge":     false,
		},
		"advanced_config": input.AdvancedConfig,
		"locations":       input.Locations,
		"caching_enabled": input.CachingEnabled,
		"hsts_subdomains": input.HSTSSubdomains,
	}

	if input.AccessListID != 0 {
		body["access_list_id"] = input.AccessListID
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
