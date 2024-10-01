package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/paradoxe35/nginxpm-operator/pkg/util"
)

const (
	LETSENCRYPT_PROVIDER = "letsencrypt"
	CUSTOM_PROVIDER      = "other"
)

type LetsEncryptCertificateMeta struct {
	LetsEncryptAgree bool    `json:"letsencrypt_agree"`
	DNSChallenge     bool    `json:"dns_challenge"`
	NginxOnline      bool    `json:"nginx_online"`
	NginxErr         *string `json:"nginx_err"`
	LetsEncryptEmail string  `json:"letsencrypt_email"`
}

type CreateLetEncryptCertificateRequest struct {
	DomainNames []string `json:"domain_names"`
	Meta        struct {
		DNSChallenge           bool   `json:"dns_challenge"`
		DNSProvider            string `json:"dns_provider"`
		DNSProviderCredentials string `json:"dns_provider_credentials"`
		LetsEncryptAgree       bool   `json:"letsencrypt_agree"`
		LetsEncryptEmail       string `json:"letsencrypt_email"`
	} `json:"meta"`
	Provider string `json:"provider"`
}

type CustomCertificateMeta struct {
	Certificate    string `json:"certificate"`
	CertificateKey string `json:"certificate_key"`
}

type certificate[K LetsEncryptCertificateMeta | CustomCertificateMeta] struct {
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

type LetsEncryptCertificate certificate[LetsEncryptCertificateMeta]

type CustomCertificate certificate[CustomCertificateMeta]

// FindExistingCertificate searches for an existing certificate matching the given domain
func (c *Client) FindLetEncryptCertificate(domain string) (*LetsEncryptCertificate, error) {
	rootDomain := util.ExtractRootDomain(domain)

	// URL encode the query parameter
	query := url.QueryEscape(rootDomain)

	resp, err := c.doRequest("GET", fmt.Sprintf("/api/nginx/certificates?query=%s", query), nil)

	if err != nil {
		return nil, fmt.Errorf("[/api/nginx/certificates] error querying certificates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("[/api/nginx/certificates] unexpected status code: %d", resp.StatusCode)
	}

	var certificates []LetsEncryptCertificate
	if err := json.NewDecoder(resp.Body).Decode(&certificates); err != nil {
		return nil, fmt.Errorf("[/api/nginx/certificates] error decoding response: %w", err)
	}

	for _, cert := range certificates {
		for _, domainName := range cert.DomainNames {
			if domainName == domain || domainName == fmt.Sprintf("*.%s", rootDomain) {
				return &cert, nil
			}
		}
	}

	return nil, nil // No matching certificate found
}

// FindCertificateByID retrieves a certificate by its ID
func (c *Client) FindLetEncryptCertificateByID(id uint16) (*LetsEncryptCertificate, error) {
	resp, err := c.doRequest("GET", "/api/nginx/certificates", nil)

	if err != nil {
		return nil, fmt.Errorf("[/api/nginx/certificates] error querying certificates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("[/api/nginx/certificates] unexpected status code: %d", resp.StatusCode)
	}

	var certificates []LetsEncryptCertificate
	if err := json.NewDecoder(resp.Body).Decode(&certificates); err != nil {
		return nil, fmt.Errorf("[/api/nginx/certificates] error decoding response: %w", err)
	}

	for _, cert := range certificates {
		if cert.ID == id {
			return &cert, nil
		}
	}

	return nil, nil // No matching certificate found
}

// LetEncryptCertificate creates a new certificate for the given domains or returns an existing one if found
func (c *Client) CreateLetEncryptCertificate(request CreateLetEncryptCertificateRequest) (*LetsEncryptCertificate, error) {
	var existingCertificate *LetsEncryptCertificate

	for i, domain := range request.DomainNames {
		cert, err := c.FindLetEncryptCertificate(domain)
		if err != nil {
			return nil, fmt.Errorf("[POST /api/nginx/certificates] error finding existing certificate for domain %s: %w", domain, err)
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
		"domain_names": request.DomainNames,
		"meta": map[string]interface{}{
			"letsencrypt_agree":        true,
			"letsencrypt_email":        request.Meta.LetsEncryptEmail,
			"dns_challenge":            request.Meta.DNSChallenge,
			"dns_provider":             request.Meta.DNSProvider,
			"dns_provider_credentials": request.Meta.DNSProviderCredentials,
		},
		"provider": LETSENCRYPT_PROVIDER,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("[POST /api/nginx/certificates] error marshaling request body: %w", err)
	}

	resp, err := c.doRequest("POST", "/api/nginx/certificates", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("[POST /api/nginx/certificates] error creating certificate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("[POST /api/nginx/certificates] unexpected status code when creating certificate: %d", resp.StatusCode)
	}

	var newCert LetsEncryptCertificate

	if err := json.NewDecoder(resp.Body).Decode(&newCert); err != nil {
		return nil, fmt.Errorf("[POST /api/nginx/certificates] error decoding response: %w", err)
	}

	return &newCert, nil
}

// DeleteCertificate deletes a certificate by its ID
func (c *Client) DeleteCertificate(id int) error {
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/nginx/certificates/%d", id), nil)
	if err != nil {
		return fmt.Errorf("[DELETE /api/nginx/certificates/%d] error deleting certificate: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[DELETE /api/nginx/certificates/%d] unexpected status code: %d", id, resp.StatusCode)
	}

	return nil
}
