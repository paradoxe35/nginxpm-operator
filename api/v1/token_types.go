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

type Secret struct {
	// SecretName is the name of the secret resource to add to the token cr
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

	// Endpoint of a Nginx Proxy manager instance
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^https?://`
	Endpoint string `json:"endpoint,omitempty"`

	// Secret resource reference to add to the token cr
	// +kubebuilder:default:={}
	// +required
	Secret Secret `json:"secret,omitempty"`

	// Expiration time of the token, value is generated from controller reconcile
	// +kubebuilder:validation:Format=date-time
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +optional
	Expires *metav1.Time `json:"expires,omitempty"`

	// Authentication Token, this is generated from controller reconcile
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +optional
	Token string `json:"token,omitempty"`
}

// TokenStatus defines the observed state of Token
type TokenStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Represents the observations of a Memcached's current state.
	// Memcached.status.conditions.type are: "Available", "Progressing", and "Degraded"
	// Memcached.status.conditions.status are one of True, False, Unknown.
	// Memcached.status.conditions.reason the value should be a CamelCase string and producers of specific
	// condition types may define expected values and meanings for this field, and whether the values
	// are considered a guaranteed API.
	// Memcached.status.conditions.Message is a human readable message indicating details about the transition.
	// For further information see: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

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