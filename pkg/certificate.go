package pkg

import (
	"encoding/json"
	"fmt"
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
