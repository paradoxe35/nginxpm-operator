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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CustomCertificateCredentialsSecret struct {
	// Name specifies the Kubernetes Secret containing the certificate data.
	// The Secret must contain the certificate and private key fields.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +required
	Name string `json:"name,omitempty"`
}

type CustomCertificateCredentials struct {
	// Secret references the Kubernetes Secret containing the SSL/TLS certificate.
	// The referenced Secret should contain the certificate chain and private key.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=object
	// +required
	Secret CustomCertificateCredentialsSecret `json:"secret,omitempty"`
}

// CustomCertificateSpec defines the desired state of CustomCertificate
type CustomCertificateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Token references the authentication token for the Nginx Proxy Manager API.
	// If not provided, the operator will search for a token named "token-nginxpm" in:
	// 1. The same namespace as this CustomCertificate
	// 2. The "nginxpm-operator-system" namespace
	// 3. The "default" namespace
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +Optional
	Token *TokenName `json:"token,omitempty"`

	// NiceName provides a human-readable display name for the certificate.
	// If not specified, the CustomCertificate resource name will be used.
	// This name appears in the Nginx Proxy Manager UI for easier identification.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:Type=string
	// +Optional
	NiceName *string `json:"niceName,omitempty"`

	// Certificate references the Kubernetes Secret containing the SSL/TLS certificate data.
	// The Secret must include both the certificate chain and the private key.
	// This certificate will be uploaded to Nginx Proxy Manager for use with proxy hosts.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=object
	// +required
	Certificate CustomCertificateCredentials `json:"certificate,omitempty"`
}

// CustomCertificateStatus defines the observed state of CustomCertificate
type CustomCertificateStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Id represents the unique identifier assigned by the Nginx Proxy Manager instance.
	// This field is populated after successful certificate upload to NPM.
	Id *int `json:"id,omitempty"`

	// ExpiresOn indicates when the SSL/TLS certificate will expire.
	// Format: ISO 8601 date-time string.
	// This value is extracted from the certificate and updated during synchronization.
	// +optional
	ExpiresOn *string `json:"expiresOn,omitempty"`

	// Status reflects the current state of the certificate in Nginx Proxy Manager.
	// Common values include "valid", "expired", "expiring_soon".
	// This field helps monitor certificate health and renewal requirements.
	// +optional
	Status *string `json:"status,omitempty"`

	// Conditions represent the current state of the CustomCertificate resource.
	// Common condition types include "Ready", "Valid", and "Synced".
	// The "Ready" condition indicates if the certificate is successfully configured in NPM.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ID",type="integer",JSONPath=".status.id"
// +kubebuilder:printcolumn:name="ExpiresOn",type="string",JSONPath=".status.expiresOn"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status"

// CustomCertificate is the Schema for the customcertificates API
type CustomCertificate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CustomCertificateSpec   `json:"spec,omitempty"`
	Status CustomCertificateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CustomCertificateList contains a list of CustomCertificate
type CustomCertificateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CustomCertificate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CustomCertificate{}, &CustomCertificateList{})
}
