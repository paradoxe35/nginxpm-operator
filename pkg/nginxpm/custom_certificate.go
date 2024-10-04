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
	"mime/multipart"
	"net/http"
)

type CustomCertificateMeta struct {
	Certificate    string `json:"certificate"`
	CertificateKey string `json:"certificate_key"`
}

type CustomCertificate certificate[CustomCertificateMeta]

type certificateValidationDate struct {
	From int64 `json:"from"`
	To   int64 `json:"to"`
}

type certificateValidationDateCert struct {
	CN     string                    `json:"cn"`
	Issuer string                    `json:"issuer"`
	Dates  certificateValidationDate `json:"dates"`
}

type CertificateValidationResponse struct {
	Certificate    certificateValidationDateCert `json:"certificate"`
	CertificateKey bool                          `json:"certificate_key"`
}

type CertificateUploadResponse struct {
	Certificate    string `json:"certificate"`
	CertificateKey string `json:"certificate_key"`
}

type CreateCustomCertificateRequest struct {
	NiceName       string `json:"nice_name"`
	Provider       string `json:"provider"`
	Certificate    []byte `json:"certificate"`
	CertificateKey []byte `json:"certificate_key"`
}

// creates an empty custom certificate for later upload of a certificate
func (c *Client) CreateEmptyCustomCertificate(name string) (*CustomCertificate, error) {
	// Create new certificate
	body := map[string]interface{}{
		"nice_name": name,
		"provider":  CUSTOM_PROVIDER,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("[CreateEmptyCustomCertificate] error marshaling request body: %w", err)
	}

	resp, err := c.doRequest("POST", "/api/nginx/certificates", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("[CreateEmptyCustomCertificate] error creating certificate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("[CreateEmptyCustomCertificate] unexpected status code when creating certificate: %d", resp.StatusCode)
	}

	newCert := new(CustomCertificate)

	if err := json.NewDecoder(resp.Body).Decode(newCert); err != nil {
		return nil, fmt.Errorf("[CreateEmptyCustomCertificate] error decoding response: %w", err)
	}

	return newCert, nil
}

func (c *Client) certificateFilesFromBytes(certificateContent, certificateKeyContent []byte) (*bytes.Buffer, *multipart.Writer, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add certificate file
	part, err := writer.CreateFormFile("certificate", "certificate.pem")
	if err != nil {
		return nil, nil, fmt.Errorf("error creating form file for certificate: %w", err)
	}
	_, err = part.Write(certificateContent)
	if err != nil {
		return nil, nil, fmt.Errorf("error writing certificate content: %w", err)
	}

	// Add certificate key file
	part, err = writer.CreateFormFile("certificate_key", "certificate_key.key")
	if err != nil {
		return nil, nil, fmt.Errorf("error creating form file for certificate key: %w", err)
	}
	_, err = part.Write(certificateKeyContent)
	if err != nil {
		return nil, nil, fmt.Errorf("error writing certificate key content: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("error closing multipart writer: %w", err)
	}

	return body, writer, nil
}

// ValidateCertificate validates a self-signed certificate and its key
func (c *Client) ValidateCustomCertificate(certificateContent, certificateKeyContent []byte) (*CertificateValidationResponse, error) {
	body, writer, err := c.certificateFilesFromBytes(certificateContent, certificateKeyContent)
	if err != nil {
		return nil, fmt.Errorf("[ValidateCertificate] err: %w", err)
	}

	req, err := http.NewRequest("POST", c.Endpoint+"/api/nginx/certificates/validate", body)
	if err != nil {
		return nil, fmt.Errorf("[ValidateCertificate] error creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[ValidateCertificate] error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"[ValidateCertificate] unexpected status code: %d, you might want to check the certificate and key content",
			resp.StatusCode,
		)
	}

	var validationResponse CertificateValidationResponse
	err = json.NewDecoder(resp.Body).Decode(&validationResponse)
	if err != nil {
		return nil, fmt.Errorf("[ValidateCertificate] error decoding response: %w", err)
	}

	return &validationResponse, nil
}

// UploadCertificate uploads a validated certificate and its key to a specific certificate ID
func (c *Client) UploadCustomCertificate(id uint16, certificateContent, certificateKeyContent []byte) (*CertificateUploadResponse, error) {
	body, writer, err := c.certificateFilesFromBytes(certificateContent, certificateKeyContent)
	if err != nil {
		return nil, fmt.Errorf("[UploadCertificate-{id}] err: %w", err)
	}

	url := fmt.Sprintf("/api/nginx/certificates/%d/upload", id)
	req, err := http.NewRequest("POST", c.Endpoint+url, body)
	if err != nil {
		return nil, fmt.Errorf("[UploadCertificate-{id}] error creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[UploadCertificate-{id}] error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[UploadCertificate-{id}] unexpected status code: %d", resp.StatusCode)
	}

	var uploadResponse CertificateUploadResponse
	err = json.NewDecoder(resp.Body).Decode(&uploadResponse)
	if err != nil {
		return nil, fmt.Errorf("[UploadCertificate-{id}] error decoding response: %w", err)
	}

	return &uploadResponse, nil
}

func (c *Client) GetCustomCertificates() ([]CustomCertificate, error) {
	resp, err := c.doRequest("GET", "/api/nginx/certificates", nil)

	if err != nil {
		return nil, fmt.Errorf("[GetCustomCertificates] error querying certificates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("[GetCustomCertificates] unexpected status code: %d", resp.StatusCode)
	}

	var certificates []CustomCertificate
	if err := json.NewDecoder(resp.Body).Decode(&certificates); err != nil {
		return nil, fmt.Errorf("[GetCustomCertificates] error decoding response: %w", err)
	}

	return certificates, nil
}

// FindCertificateByID retrieves a certificate by its ID
func (c *Client) FindCustomCertificateByID(id uint16) (*CustomCertificate, error) {
	certificates, err := c.GetCustomCertificates()
	if err != nil {
		return nil, err
	}

	for _, cert := range certificates {
		if cert.ID == id && cert.Provider == CUSTOM_PROVIDER {
			cert.Bound = false
			return &cert, nil
		}
	}

	return nil, nil // No matching certificate found
}

// FindCustomCertificateByName retrieves a certificate by its ID
func (c *Client) FindCustomCertificateByName(name string) (*CustomCertificate, error) {
	certificates, err := c.GetCustomCertificates()
	if err != nil {
		return nil, err
	}

	for _, cert := range certificates {
		if cert.NiceName == name && cert.Provider == CUSTOM_PROVIDER {
			cert.Bound = false
			return &cert, nil
		}
	}

	return nil, nil // No matching certificate found
}

// CreateCustomCertificate creates a new custom certificate
func (c *Client) CreateCustomCertificate(data CreateCustomCertificateRequest) (*CustomCertificate, error) {
	// Validate certificate and key
	_, err := c.ValidateCustomCertificate(data.Certificate, data.CertificateKey)
	if err != nil {
		return nil, fmt.Errorf("error validating certificate and key: %w", err)
	}

	// Create empty certificate
	emptyCert, err := c.CreateEmptyCustomCertificate(data.NiceName)
	if err != nil {
		return nil, fmt.Errorf("error creating empty certificate: %w", err)
	}

	// Upload certificate and key
	_, err = c.UploadCustomCertificate(emptyCert.ID, data.Certificate, data.CertificateKey)
	if err != nil {
		return nil, fmt.Errorf("error uploading certificate and key: %w", err)
	}

	// find certificate by ID
	cert, err := c.FindCustomCertificateByID(emptyCert.ID)
	if err != nil {
		return nil, fmt.Errorf("error finding certificate by ID: %w", err)
	}

	return cert, nil
}
