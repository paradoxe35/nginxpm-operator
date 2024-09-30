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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LetsEncryptCertificateSpec defines the desired state of LetsEncryptCertificate
type LetsEncryptCertificateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Token resource reference to add to the LetsEncryptCertificate, this is the created auth token
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=object
	// +required
	Token TokenName `json:"token,omitempty"`
}

// LetsEncryptCertificateStatus defines the observed state of LetsEncryptCertificate
type LetsEncryptCertificateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// LetsEncryptCertificate ID in the Nginx Proxy Manager instance
	Id *int16 `json:"id,omitempty"`

	// Whether the LetsEncryptCertificate was bound with an existing certificate
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	Bound bool `json:"bound,omitempty"`

	// Represents the observations of a LetsEncryptCertificate's current state.
	// LetsEncryptCertificate.status.conditions.type are: "Available", "Progressing", and "Degraded"
	// LetsEncryptCertificate.status.conditions.status are one of True, False, Unknown.
	// LetsEncryptCertificate.status.conditions.reason the value should be a CamelCase string and producers of specific
	// condition types may define expected values and meanings for this field, and whether the values
	// are considered a guaranteed API.
	// LetsEncryptCertificate.status.conditions.Message is a human readable message indicating details about the transition.
	// For further information see: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// LetsEncryptCertificate is the Schema for the letsencryptcertificates API
type LetsEncryptCertificate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LetsEncryptCertificateSpec   `json:"spec,omitempty"`
	Status LetsEncryptCertificateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LetsEncryptCertificateList contains a list of LetsEncryptCertificate
type LetsEncryptCertificateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LetsEncryptCertificate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LetsEncryptCertificate{}, &LetsEncryptCertificateList{})
}
