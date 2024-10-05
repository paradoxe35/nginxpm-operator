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
	"net/http/httptest"
	"testing"
	"time"

	"github.com/paradoxe35/nginxpm-operator/pkg/util"
)

func TestFindExistingLetEncryptCertificate(t *testing.T) {
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
					Provider:    LETSENCRYPT_PROVIDER,
					NiceName:    "Example Cert",
					DomainNames: []string{"example.com"},
					ExpiresOn:   now.AddDate(0, 3, 0).Format(time.RFC3339),
				},
			},
			expectedCert: &LetsEncryptCertificate{
				ID:          1,
				CreatedOn:   now.Format(time.RFC3339),
				ModifiedOn:  now.Format(time.RFC3339),
				Provider:    LETSENCRYPT_PROVIDER,
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
					Provider:    LETSENCRYPT_PROVIDER,
					NiceName:    "Example Cert",
					DomainNames: []string{"*.example.com"},
					ExpiresOn:   now.AddDate(0, 3, 0).Format(time.RFC3339),
				},
			},
			expectedCert: &LetsEncryptCertificate{
				ID:          1,
				CreatedOn:   now.Format(time.RFC3339),
				ModifiedOn:  now.Format(time.RFC3339),
				Provider:    LETSENCRYPT_PROVIDER,
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
				compareLetsEncryptCertificates(t, tt.expectedCert, cert)
			}
		})
	}
}

func TestFindLetEncryptCertificateByID(t *testing.T) {
	now := time.Now()
	serverResponse := []LetsEncryptCertificate{
		{
			ID:          1,
			CreatedOn:   now.Format(time.RFC3339),
			ModifiedOn:  now.Format(time.RFC3339),
			Provider:    LETSENCRYPT_PROVIDER,
			NiceName:    "Example Cert",
			DomainNames: []string{"example.com"},
			ExpiresOn:   now.AddDate(0, 3, 0).Format(time.RFC3339),
		},
	}

	tests := []struct {
		name           string
		certificateID  int
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
				Provider:    LETSENCRYPT_PROVIDER,
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

				compareLetsEncryptCertificates(t, tt.expectedCert, cert)
			}
		})
	}
}

func TestCreateLetEncryptCertificate(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name          string
		domains       []string
		existingCerts []LetsEncryptCertificate
		newCert       *LetsEncryptCertificate
		expectedCert  *LetsEncryptCertificate
		expectNewCert bool
		expectError   bool
	}{
		{
			name:          "New certificate creation",
			domains:       []string{"example.com", "www.example.com"},
			existingCerts: []LetsEncryptCertificate{},
			newCert: &LetsEncryptCertificate{
				ID:          1,
				CreatedOn:   now.Format(time.RFC3339),
				ModifiedOn:  now.Format(time.RFC3339),
				Provider:    LETSENCRYPT_PROVIDER,
				NiceName:    "example.com",
				DomainNames: []string{"example.com", "www.example.com"},
				ExpiresOn:   now.AddDate(0, 3, 0).Format(time.RFC3339),
			},
			expectedCert:  nil, // Will be set to newCert
			expectNewCert: true,
			expectError:   false,
		},
		{
			name:    "Existing certificate found",
			domains: []string{"existing.com"},
			existingCerts: []LetsEncryptCertificate{
				{
					ID:          2,
					CreatedOn:   now.Format(time.RFC3339),
					ModifiedOn:  now.Format(time.RFC3339),
					Provider:    LETSENCRYPT_PROVIDER,
					NiceName:    "existing.com",
					DomainNames: []string{"existing.com"},
					ExpiresOn:   now.AddDate(0, 3, 0).Format(time.RFC3339),
				},
			},
			newCert:       nil,
			expectedCert:  nil, // Will be set to existing cert
			expectNewCert: false,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				switch r.URL.Path {
				case "/api/nginx/certificates":
					if r.Method == "GET" {
						json.NewEncoder(w).Encode(tt.existingCerts)
					} else if r.Method == "POST" {
						if tt.expectNewCert {
							w.WriteHeader(http.StatusCreated)
							json.NewEncoder(w).Encode(tt.newCert)
						} else {
							w.WriteHeader(http.StatusBadRequest)
						}
					}
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			client := NewClient(server.Client(), server.URL)
			cert, err := client.CreateLetEncryptCertificate(CreateLetEncryptCertificateRequest{
				DomainNames: tt.domains,
				Meta: CreateLetEncryptCertificateRequestMeta{
					DNSChallenge:           false,
					DNSProvider:            "acmedns",
					DNSProviderCredentials: "credentials",
					LetsEncryptAgree:       true,
					LetsEncryptEmail:       "admin@example.com",
				},
			})

			if (err != nil) != tt.expectError {
				t.Fatalf("Unexpected error status: got error %v, expectError %v", err, tt.expectError)
			}

			if tt.expectNewCert {
				if cert == nil {
					t.Fatalf("Expected new certificate, got nil")
				}
				compareLetsEncryptCertificates(t, tt.newCert, cert)
			} else if len(tt.existingCerts) > 0 {
				if cert == nil {
					t.Fatalf("Expected existing certificate, got nil")
				}

				compareLetsEncryptCertificates(t, &tt.existingCerts[0], cert)
			} else if cert != nil {
				t.Errorf("Expected no certificate, got %+v", cert)
			}
		})
	}
}

func TestDeleteLetEncryptCertificate(t *testing.T) {
	tests := []struct {
		name          string
		certificateID int
		serverStatus  int
		expectError   bool
	}{
		{
			name:          "Successful deletion",
			certificateID: 1,
			serverStatus:  http.StatusOK,
			expectError:   false,
		},
		{
			name:          "Certificate not found",
			certificateID: 999,
			serverStatus:  http.StatusNotFound,
			expectError:   true,
		},
		{
			name:          "Server error",
			certificateID: 2,
			serverStatus:  http.StatusInternalServerError,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("Expected 'DELETE' request, got '%s'", r.Method)
				}

				expectedPath := fmt.Sprintf("/api/nginx/certificates/%d", tt.certificateID)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected request to '%s', got '%s'", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			client := NewClient(server.Client(), server.URL)

			err := client.DeleteCertificate(tt.certificateID)

			if (err != nil) != tt.expectError {
				t.Fatalf("Unexpected error status: got error %v, expectError %v", err, tt.expectError)
			}

			if tt.expectError && err == nil {
				t.Errorf("Expected an error, but got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}
		})
	}
}

func compareLetsEncryptCertificates(t *testing.T, expected, actual *LetsEncryptCertificate) {
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
