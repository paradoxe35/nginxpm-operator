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
	"strings"
	"testing"
	"time"
)

func TestValidateCertificate(t *testing.T) {
	tests := []struct {
		name               string
		certificateContent string
		certificateKey     string
		serverResponse     CertificateValidationResponse
		serverStatus       int
		expectError        bool
	}{
		{
			name:               "Successful validation",
			certificateContent: "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----",
			certificateKey:     "-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----",
			serverResponse: CertificateValidationResponse{
				Certificate: certificateValidationDateCert{
					CN:     "RW,",
					Issuer: "C = AU, ST = Province, L = City, O = Company Ltd, CN = example, emailAddress = admin@example.com",
					Dates:  certificateValidationDate{From: 1727986403, To: 2043346403},
				},
				CertificateKey: true,
			},
			serverStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name:               "Invalid certificate",
			certificateContent: "invalid certificate content",
			certificateKey:     "invalid key content",
			serverResponse:     CertificateValidationResponse{},
			serverStatus:       http.StatusBadRequest,
			expectError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected 'POST' request, got '%s'", r.Method)
				}

				if r.URL.Path != "/api/nginx/certificates/validate" {
					t.Errorf("Expected request to '/api/nginx/certificates/validate', got '%s'", r.URL.Path)
				}

				contentType := r.Header.Get("Content-Type")
				if !strings.HasPrefix(contentType, "multipart/form-data") {
					t.Errorf("Expected Content-Type starting with 'multipart/form-data', got '%s'", contentType)
				}

				err := r.ParseMultipartForm(10 << 20) // 10 MB
				if err != nil {
					t.Fatalf("Error parsing multipart form: %v", err)
				}

				_, _, err = r.FormFile("certificate")
				if err != nil {
					t.Errorf("Error getting certificate file: %v", err)
				}

				_, _, err = r.FormFile("certificate_key")
				if err != nil {
					t.Errorf("Error getting certificate key file: %v", err)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.serverStatus)
				if tt.serverStatus == http.StatusOK {
					json.NewEncoder(w).Encode(tt.serverResponse)
				}
			}))
			defer server.Close()

			client := NewClient(server.Client(), server.URL)

			response, err := client.ValidateCustomCertificate([]byte(tt.certificateContent), []byte(tt.certificateKey))

			if (err != nil) != tt.expectError {
				t.Fatalf("Unexpected error status: got error %v, expectError %v", err, tt.expectError)
			}

			if !tt.expectError {
				if response == nil {
					t.Fatal("Expected a response, got nil")
				}
				if response.Certificate.CN != tt.serverResponse.Certificate.CN {
					t.Errorf("Expected CN %s, got %s", tt.serverResponse.Certificate.CN, response.Certificate.CN)
				}
				if response.Certificate.Issuer != tt.serverResponse.Certificate.Issuer {
					t.Errorf("Expected Issuer %s, got %s", tt.serverResponse.Certificate.Issuer, response.Certificate.Issuer)
				}
				if response.Certificate.Dates.From != tt.serverResponse.Certificate.Dates.From {
					t.Errorf("Expected From date %d, got %d", tt.serverResponse.Certificate.Dates.From, response.Certificate.Dates.From)
				}
				if response.Certificate.Dates.To != tt.serverResponse.Certificate.Dates.To {
					t.Errorf("Expected To date %d, got %d", tt.serverResponse.Certificate.Dates.To, response.Certificate.Dates.To)
				}
				if response.CertificateKey != tt.serverResponse.CertificateKey {
					t.Errorf("Expected CertificateKey %v, got %v", tt.serverResponse.CertificateKey, response.CertificateKey)
				}
			}
		})
	}
}

func TestCreateEmptyCustomCertificate(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name            string
		certificateName string
		serverResponse  *CustomCertificate
		serverStatus    int
		expectError     bool
	}{
		{
			name:            "Successful creation",
			certificateName: "Test Custom Cert",
			serverResponse: &CustomCertificate{
				ID:          1,
				CreatedOn:   now.Format(time.RFC3339),
				ModifiedOn:  now.Format(time.RFC3339),
				Provider:    CUSTOM_PROVIDER,
				NiceName:    "Test Custom Cert",
				DomainNames: []string{},
				ExpiresOn:   now.AddDate(1, 0, 0).Format(time.RFC3339),
			},
			serverStatus: http.StatusCreated,
			expectError:  false,
		},
		{
			name:            "Server error",
			certificateName: "Error Cert",
			serverResponse:  nil,
			serverStatus:    http.StatusInternalServerError,
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected 'POST' request, got '%s'", r.Method)
				}

				if r.URL.Path != "/api/nginx/certificates" {
					t.Errorf("Expected request to '/api/nginx/certificates', got '%s'", r.URL.Path)
				}

				var requestBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&requestBody)
				if err != nil {
					t.Fatalf("Error decoding request body: %v", err)
				}

				if requestBody["nice_name"] != tt.certificateName {
					t.Errorf("Expected nice_name '%s', got '%s'", tt.certificateName, requestBody["nice_name"])
				}

				if requestBody["provider"] != CUSTOM_PROVIDER {
					t.Errorf("Expected provider '%s', got '%s'", CUSTOM_PROVIDER, requestBody["provider"])
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.serverStatus)
				if tt.serverStatus == http.StatusCreated {
					json.NewEncoder(w).Encode(tt.serverResponse)
				}
			}))
			defer server.Close()

			client := NewClient(server.Client(), server.URL)

			cert, err := client.CreateEmptyCustomCertificate(tt.certificateName)

			if (err != nil) != tt.expectError {
				t.Fatalf("Unexpected error status: got error %v, expectError %v", err, tt.expectError)
			}

			if !tt.expectError {
				if cert == nil {
					t.Fatal("Expected a certificate, got nil")
				}
				if cert.ID != tt.serverResponse.ID {
					t.Errorf("Expected ID %d, got %d", tt.serverResponse.ID, cert.ID)
				}
				if cert.NiceName != tt.serverResponse.NiceName {
					t.Errorf("Expected NiceName %s, got %s", tt.serverResponse.NiceName, cert.NiceName)
				}
				if cert.Provider != tt.serverResponse.Provider {
					t.Errorf("Expected Provider %s, got %s", tt.serverResponse.Provider, cert.Provider)
				}
			}
		})
	}
}

func TestUploadCertificate(t *testing.T) {
	tests := []struct {
		name               string
		certificateID      int
		certificateContent string
		certificateKey     string
		serverResponse     CertificateUploadResponse
		serverStatus       int
		expectError        bool
	}{
		{
			name:               "Successful upload",
			certificateID:      1,
			certificateContent: "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----",
			certificateKey:     "-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----",
			serverResponse: CertificateUploadResponse{
				Certificate:    "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----",
				CertificateKey: "-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----",
			},
			serverStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name:               "Invalid certificate ID",
			certificateID:      999,
			certificateContent: "invalid certificate content",
			certificateKey:     "invalid key content",
			serverResponse:     CertificateUploadResponse{},
			serverStatus:       http.StatusNotFound,
			expectError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected 'POST' request, got '%s'", r.Method)
				}

				expectedPath := fmt.Sprintf("/api/nginx/certificates/%d/upload", tt.certificateID)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected request to '%s', got '%s'", expectedPath, r.URL.Path)
				}

				contentType := r.Header.Get("Content-Type")
				if !strings.HasPrefix(contentType, "multipart/form-data") {
					t.Errorf("Expected Content-Type starting with 'multipart/form-data', got '%s'", contentType)
				}

				err := r.ParseMultipartForm(10 << 20) // 10 MB
				if err != nil {
					t.Fatalf("Error parsing multipart form: %v", err)
				}

				_, _, err = r.FormFile("certificate")
				if err != nil {
					t.Errorf("Error getting certificate file: %v", err)
				}

				_, _, err = r.FormFile("certificate_key")
				if err != nil {
					t.Errorf("Error getting certificate key file: %v", err)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.serverStatus)
				if tt.serverStatus == http.StatusOK {
					json.NewEncoder(w).Encode(tt.serverResponse)
				}
			}))
			defer server.Close()

			client := NewClient(server.Client(), server.URL)

			response, err := client.UploadCustomCertificate(tt.certificateID, []byte(tt.certificateContent), []byte(tt.certificateKey))

			if (err != nil) != tt.expectError {
				t.Fatalf("Unexpected error status: got error %v, expectError %v", err, tt.expectError)
			}

			if !tt.expectError {
				if response == nil {
					t.Fatal("Expected a response, got nil")
				}
				if response.Certificate != tt.serverResponse.Certificate {
					t.Errorf("Expected Certificate %s, got %s", tt.serverResponse.Certificate, response.Certificate)
				}
				if response.CertificateKey != tt.serverResponse.CertificateKey {
					t.Errorf("Expected CertificateKey %s, got %s", tt.serverResponse.CertificateKey, response.CertificateKey)
				}
			}
		})
	}
}

func TestFindCustomCertificateByID(t *testing.T) {
	now := time.Now()
	serverResponse := []CustomCertificate{
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
		serverResponse []CustomCertificate
		expectedCert   *CustomCertificate
		serverStatus   int
		expectError    bool
	}{
		{
			name:           "Existing certificate",
			certificateID:  1,
			serverResponse: serverResponse,
			expectedCert: &CustomCertificate{
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

			cert, err := client.FindCustomCertificateByID(tt.certificateID)

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

				compareCustomCertificates(t, tt.expectedCert, cert)
			}
		})
	}
}

func compareCustomCertificates(t *testing.T, expected, actual *CustomCertificate) {
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
}
