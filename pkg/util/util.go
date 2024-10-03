package util

import (
	"net/http"
	"strings"
	"time"
)

// ExtractRootDomain extracts the root domain from a given domain string.
// If the domain has two or fewer parts, it returns the entire domain.
// Otherwise, it returns the domain without the first subdomain.
func ExtractRootDomain(domain string) string {
	domain = strings.TrimSuffix(domain, ".")

	parts := strings.Split(domain, ".")
	if len(parts) <= 2 {
		return domain
	}
	return strings.Join(parts[1:], ".")
}

func NewHttpClient() *http.Client {
	return &http.Client{
		Timeout: time.Duration(45) * time.Second,
	}
}
