package pkg

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/paradoxe35/nginxpm-operator/pkg/util"
)

func TestFindExistingCertificate(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name           string
		domain         string
		serverResponse []LetsEncryptCertificate
		expectedCert   *LetsEncryptCertificate
		expectError    bool
	}{
		{
			name:   "Exact domain match",
			domain: "example.com",
			serverResponse: []LetsEncryptCertificate{
				{
					ID:          1,
					CreatedOn:   now.Format(time.RFC3339),
					ModifiedOn:  now.Format(time.RFC3339),
					Provider:    "letsencrypt",
					NiceName:    "Example Cert",
					DomainNames: []string{"example.com"},
					ExpiresOn:   now.AddDate(0, 3, 0).Format(time.RFC3339),
				},
			},
			expectedCert: &LetsEncryptCertificate{
				ID:          1,
				CreatedOn:   now.Format(time.RFC3339),
				ModifiedOn:  now.Format(time.RFC3339),
				Provider:    "letsencrypt",
				NiceName:    "Example Cert",
				DomainNames: []string{"example.com"},
				ExpiresOn:   now.AddDate(0, 3, 0).Format(time.RFC3339),
			},
			expectError: false,
		},

		{
			name:   "Exact domain DNSChallenge match",
			domain: "dns.example.com",
			serverResponse: []LetsEncryptCertificate{
				{
					ID:          1,
					CreatedOn:   now.Format(time.RFC3339),
					ModifiedOn:  now.Format(time.RFC3339),
					Provider:    "letsencrypt",
					NiceName:    "Example Cert",
					DomainNames: []string{"*.example.com"},
					ExpiresOn:   now.AddDate(0, 3, 0).Format(time.RFC3339),
				},
			},
			expectedCert: &LetsEncryptCertificate{
				ID:          1,
				CreatedOn:   now.Format(time.RFC3339),
				ModifiedOn:  now.Format(time.RFC3339),
				Provider:    "letsencrypt",
				NiceName:    "Example Cert",
				DomainNames: []string{"*.example.com"},
				ExpiresOn:   now.AddDate(0, 3, 0).Format(time.RFC3339),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected 'GET' request, got '%s'", r.Method)
				}

				expectedPath := "/api/nginx/certificates"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected request to '%s', got '%s'", expectedPath, r.URL.Path)
				}

				query := r.URL.Query().Get("query")
				expectedQuery := util.ExtractRootDomain(tt.domain)
				if query != expectedQuery {
					t.Errorf("Expected query '%s', got '%s'", expectedQuery, query)
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := NewClient(server.Client(), server.URL)

			cert, err := client.FindLetEncryptCertificate(tt.domain)

			if (err != nil) != tt.expectError {
				t.Fatalf("Unexpected error status: got error %v, expectError %v", err, tt.expectError)
			}

			if tt.expectedCert == nil {
				if cert != nil {
					t.Errorf("Expected no certificate, got %+v", cert)
				}
			} else {
				if cert == nil {
					t.Fatalf("Expected certificate, got nil")
				}
				compareCertificates(t, tt.expectedCert, cert)
			}
		})
	}
}

func TestFindCertificateByID(t *testing.T) {
	now := time.Now()
	serverResponse := []LetsEncryptCertificate{
		{
			ID:          1,
			CreatedOn:   now.Format(time.RFC3339),
			ModifiedOn:  now.Format(time.RFC3339),
			Provider:    "letsencrypt",
			NiceName:    "Example Cert",
			DomainNames: []string{"example.com"},
			ExpiresOn:   now.AddDate(0, 3, 0).Format(time.RFC3339),
		},
	}

	tests := []struct {
		name           string
		certificateID  uint16
		serverResponse []LetsEncryptCertificate
		expectedCert   *LetsEncryptCertificate
		serverStatus   int
		expectError    bool
	}{
		{
			name:           "Existing certificate",
			certificateID:  1,
			serverResponse: serverResponse,
			expectedCert: &LetsEncryptCertificate{
				ID:          1,
				CreatedOn:   now.Format(time.RFC3339),
				ModifiedOn:  now.Format(time.RFC3339),
				Provider:    "letsencrypt",
				NiceName:    "Example Cert",
				DomainNames: []string{"example.com"},
				ExpiresOn:   now.AddDate(0, 3, 0).Format(time.RFC3339),
			},
			serverStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name:           "Non-existent certificate",
			certificateID:  999,
			expectedCert:   nil,
			serverResponse: serverResponse,
			serverStatus:   http.StatusOK,
			expectError:    false,
		},
		{
			name:           "Server error",
			certificateID:  2,
			expectedCert:   nil,
			serverResponse: serverResponse,
			serverStatus:   http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected 'GET' request, got '%s'", r.Method)
				}

				expectedPath := "/api/nginx/certificates"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected request to '%s', got '%s'", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.serverStatus)
				if tt.serverResponse != nil {
					json.NewEncoder(w).Encode(tt.serverResponse)
				}
			}))
			defer server.Close()

			client := NewClient(server.Client(), server.URL)

			cert, err := client.FindLetEncryptCertificateByID(tt.certificateID)

			if (err != nil) != tt.expectError {
				t.Log(tt.certificateID)
				t.Fatalf("Unexpected error status: got error %v, expectError %v", err, tt.expectError)
			}

			if tt.expectedCert == nil {
				if cert != nil {
					t.Errorf("Expected no certificate, got %+v", cert)
				}
			} else {
				if cert == nil {
					t.Fatalf("Expected certificate, got nil")
				}

				if cert.ID != tt.expectedCert.ID {
					t.Errorf("Expected certificate ID %d, got %d", tt.expectedCert.ID, cert.ID)
				}

				compareCertificates(t, tt.expectedCert, cert)
			}
		})
	}
}

func compareCertificates(t *testing.T, expected, actual *LetsEncryptCertificate) {
	if expected.ID != actual.ID {
		t.Errorf("Expected certificate ID %d, got %d", expected.ID, actual.ID)
	}
	if expected.CreatedOn != actual.CreatedOn {
		t.Errorf("Expected CreatedOn %s, got %s", expected.CreatedOn, actual.CreatedOn)
	}
	if expected.ModifiedOn != actual.ModifiedOn {
		t.Errorf("Expected ModifiedOn %s, got %s", expected.ModifiedOn, actual.ModifiedOn)
	}
	if expected.Provider != actual.Provider {
		t.Errorf("Expected Provider %s, got %s", expected.Provider, actual.Provider)
	}
	if expected.NiceName != actual.NiceName {
		t.Errorf("Expected NiceName %s, got %s", expected.NiceName, actual.NiceName)
	}
	if expected.ExpiresOn != actual.ExpiresOn {
		t.Errorf("Expected ExpiresOn %s, got %s", expected.ExpiresOn, actual.ExpiresOn)
	}
	if len(expected.DomainNames) != len(actual.DomainNames) {
		t.Errorf("Expected %d domain names, got %d", len(expected.DomainNames), len(actual.DomainNames))
	} else {
		for i, dn := range expected.DomainNames {
			if dn != actual.DomainNames[i] {
				t.Errorf("Expected domain name %s, got %s", dn, actual.DomainNames[i])
			}
		}
	}
}
