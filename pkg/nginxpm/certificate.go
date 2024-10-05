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
	"slices"
)

const (
	LETSENCRYPT_PROVIDER = "letsencrypt"
	CUSTOM_PROVIDER      = "other"
)

type certificate[K LetsEncryptCertificateMeta | CustomCertificateMeta | interface{}] struct {
	ID          uint16   `json:"id"`
	CreatedOn   string   `json:"created_on"`
	ModifiedOn  string   `json:"modified_on"`
	Provider    string   `json:"provider"`
	NiceName    string   `json:"nice_name"`
	DomainNames []string `json:"domain_names"`
	ExpiresOn   string   `json:"expires_on"`
	Meta        K        `json:"meta"`
	Bound       bool     `json:"bound"`
}

type Certificate certificate[interface{}]

// GetCertificates returns a list of certificates from the API
func (c *Client) GetCertificates() ([]Certificate, error) {
	resp, err := c.doRequest("GET", "/api/nginx/certificates", nil)

	if err != nil {
		return nil, fmt.Errorf("[GetCertificates] error querying certificates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("[GetCertificates] unexpected status code: %d", resp.StatusCode)
	}

	var certificates []Certificate
	if err := json.NewDecoder(resp.Body).Decode(&certificates); err != nil {
		return nil, fmt.Errorf("[GetCertificates] error decoding response: %w", err)
	}

	return certificates, nil
}

func (c *Client) FindCertificateByID(id uint16) (*Certificate, error) {
	certificates, err := c.GetCertificates()
	if err != nil {
		return nil, err
	}

	for _, cert := range certificates {
		if cert.ID == id {
			cert.Bound = false
			return &cert, nil
		}
	}

	return nil, nil // No matching certificate found
}

// FindCertificateByDomain searches for an existing certificate matching the given domain
// It will just match the first domain name in the list
func (c *Client) FindCertificateByDomain(domains []string) (*Certificate, error) {
	if len(domains) == 0 {
		return nil, fmt.Errorf("[FindCertificateByDomain] no domains provided")
	}

	domain := domains[0]

	certificates, err := c.GetCertificates()
	if err != nil {
		return nil, err
	}

	for _, cert := range certificates {
		if slices.Contains(cert.DomainNames, domain) {
			cert.Bound = false
			return &cert, nil
		}
	}

	return nil, nil // No matching certificate found
}
