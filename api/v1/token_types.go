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

// This is used by other resources
type TokenName struct {
	// Name specifies the Token resource to reference.
	// Used by other resources to authenticate with Nginx Proxy Manager.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +required
	Name string `json:"name,omitempty"`

	// Namespace of the Token resource.
	// If not specified, uses the same namespace as the referencing resource.
	// Must follow Kubernetes namespace naming conventions.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^[a-z]([-a-z0-9]*[a-z0-9])?$`
	// +optional
	Namespace *string `json:"namespace,omitempty"`
}

// SecretData is the data of the secret resource
type SecretData struct {
	// Identity is the authentication username or email for Nginx Proxy Manager.
	// Typically the admin email address used to log into NPM.
	// This field should be stored in a Kubernetes Secret.
	Identity string `json:"identity"`

	// Secret is the authentication password for Nginx Proxy Manager.
	// The password associated with the Identity for NPM login.
	// This sensitive field must be stored in a Kubernetes Secret.
	Secret string `json:"secret"`
}

type Secret struct {
	// SecretName references the Kubernetes Secret containing NPM credentials.
	// The Secret must contain "identity" and "secret" fields.
	// These credentials are used to authenticate with the NPM API.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +required
	SecretName string `json:"secretName"`
}

// TokenSpec defines the desired state of Token
type TokenSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Endpoint is the base URL of the Nginx Proxy Manager instance.
	// Format: "http(s)://hostname:port" (e.g., "https://npm.example.com:81").
	// This is where the operator will send API requests.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^(https?):\/\/([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}(:[0-9]{1,5})?$`
	// +required
	Endpoint string `json:"endpoint,omitempty"`

	// Secret references the Kubernetes Secret containing authentication credentials.
	// The Secret must include "identity" and "secret" data fields.
	// These credentials are used to obtain and refresh NPM API tokens.
	// +required
	Secret Secret `json:"secret,omitempty"`
}

// TokenStatus defines the observed state of Token
type TokenStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Token contains the JWT authentication token from Nginx Proxy Manager.
	// This token is automatically generated and refreshed by the operator.
	// Used internally for API authentication - do not modify manually.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +optional
	Token *string `json:"token,omitempty"`

	// Expires indicates when the current JWT token will expire.
	// The operator automatically refreshes tokens before expiration.
	// Format: Kubernetes metav1.Time (RFC3339).
	// +optional
	Expires *metav1.Time `json:"expires,omitempty"`

	// Conditions represent the current state of the Token resource.
	// Common condition types include "Ready", "Authenticated", and "TokenExpiring".
	// The "Ready" condition indicates if the token is valid and usable for API calls.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Secret",type="string",JSONPath=".spec.secret.secretName"
// +kubebuilder:printcolumn:name="Expires",type="string",JSONPath=".status.expires"

// Token is the Schema for the tokens API
type Token struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TokenSpec   `json:"spec,omitempty"`
	Status TokenStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TokenList contains a list of Token
type TokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Token `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Token{}, &TokenList{})
}
