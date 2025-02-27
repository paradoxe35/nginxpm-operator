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

type AccessListAuthorization struct {
	// Username to be used for authentication with the access list service.
	// Must be between 1 and 255 characters.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +required
	Username string `json:"username"`

	// Password to be used for authentication with the access list service.
	// Must be between 1 and 255 characters.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +required
	Password string `json:"password"`
}

type AccessListClient struct {
	// Address (IPv4 IP/SUBNET) for authentication use
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(\/([0-9]|[1-2][0-9]|3[0-2]))?$`
	// +required
	Address string `json:"address,omitempty"`

	// Directive for Authentication Use
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=allow;deny
	// +required
	Directive string `json:"directive,omitempty"`
}

// AccessListSpec defines the desired state of AccessList.
type AccessListSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Access list name that will be displayed in the Nginx Proxy Manager instance
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +required
	Name string `json:"name,omitempty"`

	// If set true, allow access if at least one condition is met when multiple authentication or access control methods are defined.
	// +kubebuilder:validation:Enum=true;false
	// +kubebuilder:validation:Default=false
	SatisfyAny bool `json:"satisfyAny,omitempty"`

	// Authorization to host should only be enabled if the host has basic authentication enabled.
	// +kubebuilder:validation:Enum=true;false
	// +kubebuilder:validation:Default=false
	PassAuth bool `json:"passAuth,omitempty"`

	// Basic Authorization via Nginx HTTP Basic Authentication (https://nginx.org/en/docs/http/ngx_http_auth_basic_module.html)
	// +kubebuilder:validation:Type=array
	// +optional
	Authorizations []AccessListAuthorization `json:"authorizations,omitempty"`

	// IP Address Whitelist/Blacklist via Nginx HTTP Access (https://nginx.org/en/docs/http/ngx_http_access_module.html)
	// +kubebuilder:validation:Type=array
	// +optional
	Clients []AccessListClient `json:"clients,omitempty"`
}

// AccessListStatus defines the observed state of AccessList.
type AccessListStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// AccessList ID in the Nginx Proxy Manager instance
	Id *int `json:"id,omitempty"`

	// Number of proxy hosts associated with this AccessList
	// +kubebuilder:validation:Minimum=0
	ProxyHostCount int `json:"proxyHostCount,omitempty"`

	// Represents the observations of a AccessListStatus's current state.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ID",type="integer",JSONPath=".status.id"
// +kubebuilder:printcolumn:name="Name",type="string",JSONPath=".spec.name"
// +kubebuilder:printcolumn:name="Proxy Host Count",type="string",JSONPath=".status.proxyHostCount"

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
