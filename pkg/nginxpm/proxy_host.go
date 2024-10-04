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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// ProxyHost represents the structure of a proxy host as returned by the API
type ProxyHost struct {
	ID                    uint16     `json:"id"`
	CreatedOn             time.Time  `json:"created_on"`
	ModifiedOn            time.Time  `json:"modified_on"`
	DomainNames           []string   `json:"domain_names"`
	ForwardHost           string     `json:"forward_host"`
	ForwardPort           int        `json:"forward_port"`
	AccessListID          int        `json:"access_list_id"`
	CertificateID         int        `json:"certificate_id"`
	SSLForced             int        `json:"ssl_forced"`
	CachingEnabled        int        `json:"caching_enabled"`
	BlockExploits         int        `json:"block_exploits"`
	AdvancedConfig        string     `json:"advanced_config"`
	Meta                  ProxyMeta  `json:"meta"`
	AllowWebsocketUpgrade int        `json:"allow_websocket_upgrade"`
	HTTP2Support          int        `json:"http2_support"`
	ForwardScheme         string     `json:"forward_scheme"`
	Enabled               int        `json:"enabled"`
	Locations             []Location `json:"locations"`
	HSTSEnabled           int        `json:"hsts_enabled"`
	HSTSSubdomains        int        `json:"hsts_subdomains"`
	IPv6                  bool       `json:"ipv6"`
}

// ProxyMeta represents the meta information for a proxy host
type ProxyMeta struct {
	LetsEncryptAgree bool    `json:"letsencrypt_agree"`
	DNSChallenge     bool    `json:"dns_challenge"`
	NginxOnline      bool    `json:"nginx_online"`
	NginxErr         *string `json:"nginx_err"`
}

// Location represents a location configuration for a proxy host
type Location struct {
	Path           string `json:"path"`
	AdvancedConfig string `json:"advanced_config"`
	ForwardScheme  string `json:"forward_scheme"`
	ForwardHost    string `json:"forward_host"`
	ForwardPort    int    `json:"forward_port"`
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
func (c *Client) FindProxyHostByID(id uint16) (*ProxyHost, error) {
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
