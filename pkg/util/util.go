package util

import "strings"

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
