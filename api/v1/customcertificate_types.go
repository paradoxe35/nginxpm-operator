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
	// Name of the secret resource
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +required
	Name string `json:"name,omitempty"`
}

type CustomCertificateCredentials struct {
	// Secret resource holds certificate values
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=object
	// +required
	Secret CustomCertificateCredentialsSecret `json:"secret,omitempty"`
}

// CustomCertificateSpec defines the desired state of CustomCertificate
type CustomCertificateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Token resource, if not provided, the operator will try to find a token with `token-nginxpm` name in the same namespace as the proxyhost is created or in the `nginxpm-operator-system` namespace or in the `default` namespace
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +Optional
	Token *TokenName `json:"token,omitempty"`

	// NiceName of the CustomCertificate (If not provided, the resource name will be used)
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:Type=string
	// +Optional
	NiceName *string `json:"niceName,omitempty"`

	// Certificate credential secret name
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=object
	// +required
	Certificate CustomCertificateCredentials `json:"certificate,omitempty"`
}

// CustomCertificateStatus defines the observed state of CustomCertificate
type CustomCertificateStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	// CustomCertificateStatus ID from remote  Nginx Proxy Manager instance
	Id *int `json:"id,omitempty"`

	// Expiration time of the certificate
	// +optional
	ExpiresOn *string `json:"expiresOn,omitempty"`

	// Status of the certificate
	// +optional
	Status *string `json:"status,omitempty"`

	// Represents the observations of a CustomCertificateStatus's current state.
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
