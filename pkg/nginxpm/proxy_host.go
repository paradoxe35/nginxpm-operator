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
	"net/http"
	"net/url"
	"time"
)

// ProxyHost represents the structure of a proxy host as returned by the API
type ProxyHost struct {
	ID                    int                 `json:"id"`
	CreatedOn             time.Time           `json:"created_on"`
	ModifiedOn            time.Time           `json:"modified_on"`
	DomainNames           []string            `json:"domain_names"`
	ForwardHost           string              `json:"forward_host"`
	ForwardPort           int                 `json:"forward_port"`
	AccessListID          int                 `json:"access_list_id"`
	CertificateID         int                 `json:"certificate_id"`
	SSLForced             int                 `json:"ssl_forced"`
	CachingEnabled        int                 `json:"caching_enabled"`
	BlockExploits         int                 `json:"block_exploits"`
	AdvancedConfig        string              `json:"advanced_config"`
	Meta                  ProxyHostMeta       `json:"meta"`
	Bound                 bool                `json:"bound"`
	AllowWebsocketUpgrade int                 `json:"allow_websocket_upgrade"`
	HTTP2Support          int                 `json:"http2_support"`
	ForwardScheme         string              `json:"forward_scheme"`
	Enabled               int                 `json:"enabled"`
	Locations             []ProxyHostLocation `json:"locations"`
	HSTSEnabled           int                 `json:"hsts_enabled"`
	HSTSSubdomains        int                 `json:"hsts_subdomains"`
	IPv6                  bool                `json:"ipv6"`
}

// ProxyMeta represents the meta information for a proxy host
type ProxyHostMeta struct {
	LetsEncryptAgree bool    `json:"letsencrypt_agree"`
	DNSChallenge     bool    `json:"dns_challenge"`
	NginxOnline      bool    `json:"nginx_online"`
	NginxErr         *string `json:"nginx_err"`
}

// Location represents a location configuration for a proxy host
type ProxyHostLocation struct {
	Path           string `json:"path"`
	AdvancedConfig string `json:"advanced_config"`
	ForwardScheme  string `json:"forward_scheme"`
	ForwardHost    string `json:"forward_host"`
	ForwardPort    int    `json:"forward_port"`
}

type CreateProxyHostInput struct {
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
}

// DeleteProxyHost deletes a certificate by its ID
func (c *Client) DeleteProxyHost(id int) error {
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/nginx/proxy-hosts/%d", id), nil)
	if err != nil {
		return fmt.Errorf("[DeleteProxyHost %d] error deleting proxy host: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[DeleteProxyHost %d] unexpected status code: %d", id, resp.StatusCode)
	}

	return nil
}

// FindProxyHostByDomain searches for an existing proxy host matching the given domains
func (c *Client) FindProxyHostByDomain(domains []string) (*ProxyHost, error) {
	if len(domains) == 0 {
		return nil, fmt.Errorf("[FindExistingProxyHost] no domains provided")
	}

	fDomain := domains[0]
	query := url.QueryEscape(fDomain)

	resp, err := c.doRequest("GET", fmt.Sprintf("/api/nginx/proxy-hosts?query=%s", query), nil)
	if err != nil {
		return nil, fmt.Errorf("[FindExistingProxyHost] error querying proxy hosts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("[FindExistingProxyHost] unexpected status code: %d", resp.StatusCode)
	}

	var hosts []ProxyHost
	err = json.NewDecoder(resp.Body).Decode(&hosts)
	if err != nil {
		return nil, fmt.Errorf("[FindExistingProxyHost] error decoding response: %w", err)
	}

	for _, host := range hosts {
		if len(host.DomainNames) != len(domains) {
			continue
		}

		validHost := true
		for _, domain := range domains {
			found := false
			for _, hostDomain := range host.DomainNames {
				if hostDomain == domain {
					found = true
					break
				}
			}
			if !found {
				validHost = false
				break
			}
		}

		if validHost {
			return &host, nil
		}
	}

	return nil, nil // No matching proxy host found
}

// FindProxyHostByID searches for an existing proxy host matching the given domains
func (c *Client) FindProxyHostByID(id int) (*ProxyHost, error) {
	resp, err := c.doRequest("GET", "/api/nginx/proxy-hosts", nil)
	if err != nil {
		return nil, fmt.Errorf("[FindProxyHostByID] error querying proxy hosts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("[FindProxyHostByID] unexpected status code: %d", resp.StatusCode)
	}

	var hosts []ProxyHost
	err = json.NewDecoder(resp.Body).Decode(&hosts)
	if err != nil {
		return nil, fmt.Errorf("[FindProxyHostByID] error decoding response: %w", err)
	}

	for _, host := range hosts {
		if host.ID == id {
			return &host, nil
		}
	}

	return nil, nil // No matching proxy host found
}

func proxyHostRequestBody(input CreateProxyHostInput) map[string]interface{} {
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

	return body
}

// CreateProxyHost creates a new proxy host
func (c *Client) CreateProxyHost(input CreateProxyHostInput) (*ProxyHost, error) {
	body := proxyHostRequestBody(input)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("[CreateProxyHost] error marshaling request body: %w", err)
	}

	resp, err := c.doRequest("POST", "/api/nginx/proxy-hosts", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("[CreateProxyHost] error creating proxy host: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("[CreateProxyHost] unexpected status code when creating proxy host: %d", resp.StatusCode)
	}

	var newProxyHost ProxyHost
	err = json.NewDecoder(resp.Body).Decode(&newProxyHost)
	if err != nil {
		return nil, fmt.Errorf("[CreateProxyHost] error decoding response: %w", err)
	}

	return &newProxyHost, nil
}

// UpdateProxyHost updates an existing proxy host
func (c *Client) UpdateProxyHost(id int, input CreateProxyHostInput) (*ProxyHost, error) {
	body := proxyHostRequestBody(input)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("[UpdateProxyHost] error marshaling request body: %w", err)
	}

	resp, err := c.doRequest("PUT", fmt.Sprintf("/api/nginx/proxy-hosts/%d", id), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("[UpdateProxyHost] error updating proxy host: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("[UpdateProxyHost] unexpected status code when updating proxy host: %d", resp.StatusCode)
	}

	var updatedProxyHost ProxyHost
	err = json.NewDecoder(resp.Body).Decode(&updatedProxyHost)
	if err != nil {
		return nil, fmt.Errorf("[UpdateProxyHost] error decoding response: %w", err)
	}

	return &updatedProxyHost, nil
}
