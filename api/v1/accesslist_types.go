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

type AccessListAuthorization struct {
	// Username defines the authentication username for HTTP Basic Authentication.
	// This username will be required when accessing the protected resources.
	// Must be between 1 and 255 characters in length.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +required
	Username string `json:"username"`

	// Password defines the authentication password for HTTP Basic Authentication.
	// This password will be paired with the username for authentication.
	// Must be between 1 and 255 characters in length.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +required
	Password string `json:"password"`
}

type AccessListClient struct {
	// Address specifies the IPv4 address or CIDR subnet for IP-based access control.
	// Format: Single IP (e.g., "192.168.1.1") or CIDR notation (e.g., "192.168.0.0/24").
	// Used in conjunction with the Directive field to allow or deny access.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(\/([0-9]|[1-2][0-9]|3[0-2]))?$`
	// +required
	Address string `json:"address,omitempty"`

	// Directive determines the access control action for the specified address.
	// "allow" permits access from the address, "deny" blocks access from the address.
	// Used with Address field to implement IP-based access control.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=allow;deny
	// +required
	Directive string `json:"directive,omitempty"`
}

// AccessListSpec defines the desired state of AccessList.
type AccessListSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Token references the authentication token for the Nginx Proxy Manager API.
	// If not provided, the operator will search for a token named "token-nginxpm" in:
	// 1. The same namespace as this AccessList
	// 2. The "nginxpm-operator-system" namespace
	// 3. The "default" namespace
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +Optional
	Token *TokenName `json:"token,omitempty"`

	// SatisfyAny controls how multiple access control methods are evaluated.
	// When true: Access is granted if ANY condition is met (logical OR).
	// When false: Access requires ALL conditions to be met (logical AND).
	// This applies when both IP restrictions and Basic Auth are configured.
	// +kubebuilder:validation:Enum=true;false
	// +kubebuilder:validation:Default=false
	SatisfyAny bool `json:"satisfyAny,omitempty"`

	// PassAuth determines whether to pass authentication headers to the upstream server.
	// When true: Basic Auth credentials are forwarded to the proxied server.
	// When false: Authentication is handled only by Nginx, credentials are not forwarded.
	// Enable this only if the upstream service also requires the authentication headers.
	// +kubebuilder:validation:Enum=true;false
	// +kubebuilder:validation:Default=false
	PassAuth bool `json:"passAuth,omitempty"`

	// Authorizations defines the list of username/password pairs for HTTP Basic Authentication.
	// These credentials will be required to access resources protected by this AccessList.
	// Based on Nginx HTTP Basic Authentication module.
	// +kubebuilder:validation:Type=array
	// +optional
	Authorizations []AccessListAuthorization `json:"authorizations,omitempty"`

	// Clients defines IP-based access control rules using allow/deny directives.
	// Each entry specifies an IP address or subnet with an associated action.
	// Rules are evaluated in order. Based on Nginx HTTP Access module.
	// +kubebuilder:validation:Type=array
	// +optional
	Clients []AccessListClient `json:"clients,omitempty"`
}

// AccessListStatus defines the observed state of AccessList.
type AccessListStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Id represents the unique identifier assigned by the Nginx Proxy Manager instance.
	// This field is populated after successful creation/synchronization with NPM.
	Id *int `json:"id,omitempty"`

	// ProxyHostCount indicates the number of ProxyHost resources currently using this AccessList.
	// This helps track AccessList usage and prevent accidental deletion of in-use lists.
	// +kubebuilder:validation:Default=0
	ProxyHostCount int `json:"proxyHostCount,omitempty"`

	// Conditions represent the current state of the AccessList resource.
	// Common condition types include "Ready", "Synced", and "Error".
	// The "Ready" condition indicates if the AccessList is successfully configured in NPM.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ID",type="integer",JSONPath=".status.id"
// +kubebuilder:printcolumn:name="Name",type="string",JSONPath=".spec.name"
// +kubebuilder:printcolumn:name="Proxy Host Count",type="integer",JSONPath=".status.proxyHostCount"

// AccessList is the Schema for the accesslists API.
type AccessList struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AccessListSpec   `json:"spec,omitempty"`
	Status AccessListStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AccessListList contains a list of AccessList.
type AccessListList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AccessList `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AccessList{}, &AccessListList{})
}
