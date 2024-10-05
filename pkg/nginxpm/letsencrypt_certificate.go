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

	"github.com/paradoxe35/nginxpm-operator/pkg/util"
)

type LetsEncryptCertificateMeta struct {
	LetsEncryptAgree bool    `json:"letsencrypt_agree"`
	DNSChallenge     bool    `json:"dns_challenge"`
	NginxOnline      bool    `json:"nginx_online"`
	NginxErr         *string `json:"nginx_err"`
	LetsEncryptEmail string  `json:"letsencrypt_email"`
}

type CreateLetEncryptCertificateRequestMeta struct {
	DNSChallenge           bool   `json:"dns_challenge"`
	DNSProvider            string `json:"dns_provider"`
	DNSProviderCredentials string `json:"dns_provider_credentials"`
	LetsEncryptAgree       bool   `json:"letsencrypt_agree"`
	LetsEncryptEmail       string `json:"letsencrypt_email"`
}

type CreateLetEncryptCertificateRequest struct {
	DomainNames []string                               `json:"domain_names"`
	Meta        CreateLetEncryptCertificateRequestMeta `json:"meta"`
}

type LetsEncryptCertificate certificate[LetsEncryptCertificateMeta]

// FindExistingCertificate searches for an existing certificate matching the given domain
func (c *Client) FindLetEncryptCertificate(domain string) (*LetsEncryptCertificate, error) {
	rootDomain := util.ExtractRootDomain(domain)

	// URL encode the query parameter
	query := url.QueryEscape(rootDomain)

	resp, err := c.doRequest("GET", fmt.Sprintf("/api/nginx/certificates?query=%s", query), nil)

	if err != nil {
		return nil, fmt.Errorf("[FindLetEncryptCertificate] error querying certificates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("[FindLetEncryptCertificate] unexpected status code: %d", resp.StatusCode)
	}

	var certificates []LetsEncryptCertificate
	if err := json.NewDecoder(resp.Body).Decode(&certificates); err != nil {
		return nil, fmt.Errorf("[FindLetEncryptCertificate] error decoding response: %w", err)
	}

	for _, cert := range certificates {
		for _, domainName := range cert.DomainNames {
			// Match the domain name with the given domain or with a wildcard domain
			matchedDomain := domainName == domain || domainName == fmt.Sprintf("*.%s", rootDomain)

			if matchedDomain && cert.Provider == LETSENCRYPT_PROVIDER {
				cert.Bound = false
				return &cert, nil
			}
		}
	}

	return nil, nil // No matching certificate found
}

// FindCertificateByID retrieves a certificate by its ID
func (c *Client) FindLetEncryptCertificateByID(id int) (*LetsEncryptCertificate, error) {
	resp, err := c.doRequest("GET", "/api/nginx/certificates", nil)

	if err != nil {
		return nil, fmt.Errorf("[FindLetEncryptCertificateByID] error querying certificates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("[FindLetEncryptCertificateByID] unexpected status code: %d", resp.StatusCode)
	}

	var certificates []LetsEncryptCertificate
	if err := json.NewDecoder(resp.Body).Decode(&certificates); err != nil {
		return nil, fmt.Errorf("[FindLetEncryptCertificateByID] error decoding response: %w", err)
	}

	for _, cert := range certificates {
		if cert.ID == id && cert.Provider == LETSENCRYPT_PROVIDER {
			cert.Bound = false
			return &cert, nil
		}
	}

	return nil, nil // No matching certificate found
}

// LetEncryptCertificate creates a new certificate for the given domains or returns an existing one if found
func (c *Client) CreateLetEncryptCertificate(data CreateLetEncryptCertificateRequest) (*LetsEncryptCertificate, error) {
	var existingCertificate *LetsEncryptCertificate

	for i, domain := range data.DomainNames {
		cert, err := c.FindLetEncryptCertificate(domain)
		if err != nil {
			return nil, fmt.Errorf("[CreateLetEncryptCertificate] error finding existing certificate for domain %s: %w", domain, err)
		}

		if cert != nil && existingCertificate != nil && existingCertificate.ID != cert.ID && i > 0 {
			existingCertificate = nil
			break
		}

		existingCertificate = cert
	}

	if existingCertificate != nil {
		// Let users know that the certificate is already bound to an existing certificate
		existingCertificate.Bound = true
		return existingCertificate, nil
	}

	// Create new certificate
	body := map[string]interface{}{
		"domain_names": data.DomainNames,
		"meta": map[string]interface{}{
			"letsencrypt_agree":        true,
			"letsencrypt_email":        data.Meta.LetsEncryptEmail,
			"dns_challenge":            data.Meta.DNSChallenge,
			"dns_provider":             data.Meta.DNSProvider,
			"dns_provider_credentials": data.Meta.DNSProviderCredentials,
		},
		"provider": LETSENCRYPT_PROVIDER,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("[CreateLetEncryptCertificate] error marshaling request body: %w", err)
	}

	resp, err := c.doRequest("POST", "/api/nginx/certificates", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("[CreateLetEncryptCertificate] error creating certificate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("[CreateLetEncryptCertificate] unexpected status code when creating certificate: %d", resp.StatusCode)
	}

	newCert := new(LetsEncryptCertificate)

	if err := json.NewDecoder(resp.Body).Decode(newCert); err != nil {
		return nil, fmt.Errorf("[CreateLetEncryptCertificate] error decoding response: %w", err)
	}

	newCert.Bound = false

	return newCert, nil
}

// DeleteCertificate deletes a certificate by its ID
func (c *Client) DeleteCertificate(id int) error {
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/nginx/certificates/%d", id), nil)
	if err != nil {
		return fmt.Errorf("[DeleteCertificate %d] error deleting certificate: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[DeleteCertificate %d] unexpected status code: %d", id, resp.StatusCode)
	}

	return nil
}
