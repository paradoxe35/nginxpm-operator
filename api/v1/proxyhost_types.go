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

// +kubebuilder:validation:Required
// +kubebuilder:validation:Type=string
// +kubebuilder:validation:Pattern=`^(\*\.)?[a-zA-Z0-9-]+(\.[a-zA-Z0-9-]+)*\.[a-zA-Z]{2,}$`
// +required
type DomainName string

type SslCustomCertificate struct {
	// Name of the custom certificate resource
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +required
	Name string `json:"name,omitempty"`

	// Namespace of the custom certificate resource
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^[a-z]([-a-z0-9]*[a-z0-9])?$`
	// +optional
	Namespace *string `json:"namespace,omitempty"`
}

type SslLetsEncryptCertificate struct {
	// Name of the letsencrypt certificate resource
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +required
	Name string `json:"name,omitempty"`

	// Namespace of the letsencrypt certificate resource
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^[a-z]([-a-z0-9]*[a-z0-9])?$`
	// +optional
	Namespace *string `json:"namespace,omitempty"`
}

type Ssl struct {
	// When true, will request a certificate from Let's Encrypt automatically
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	AutoCertificateRequest bool `json:"autoCertificateRequest,omitempty"`

	// Letsencrypt Certificate name created or managed by the letsencryptCertificate resource
	// If CustomCertificate is provided and LetsencryptCertificate is not provided, the CustomCertificate will be prioritized
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	LetsEncryptCertificate *SslLetsEncryptCertificate `json:"letsEncryptCertificate,omitempty"`

	// Custom Certificate name created or managed by the customCertificate resource
	// If CustomCertificate is provided and LetsencryptCertificate is not provided, the CustomCertificate will be prioritized
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	CustomCertificate *SslCustomCertificate `json:"customCertificate,omitempty"`

	// Bind existing certificate id to the proxyhost
	// This will be considered only if CustomCertificate or LetsencryptCertificate is not provided
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=integer
	// +optional
	CertificateId *int `json:"certificateId,omitempty"`

	// LetsEncrypt Email address to request a certificate for
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	// +optional
	LetsEncryptEmail *string `json:"letsEncryptEmail,omitempty"`

	// Force SSL https, redirect http to https. default is true
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	SslForced bool `json:"sslForced,omitempty"`

	// Enable http2 support, default is true
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	Http2Support bool `json:"http2Support,omitempty"`

	// Enable HSTS, default is false
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	HstsEnabled bool `json:"hstsEnabled,omitempty"`

	// Enable HSTS subdomains, default is false
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	HstsSubdomains bool `json:"hstsSubdomains,omitempty"`
}

type ForwardHost struct {
	// The host to forward to (This must be a valid DNS name or IP address)
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^((([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)\.)*[a-zA-Z]{2,63}|((25[0-5]|2[0-4][0-9]|[0-1]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[0-1]?[0-9][0-9]?)|(([0-9a-fA-F]{1,4}:){7}([0-9a-fA-F]{1,4}|:))|(::([0-9a-fA-F]{1,4}:){0,6}[0-9a-fA-F]{1,4}))$`
	// +required
	HostName string `json:"hostName,omitempty"`

	// Service Target Port is the port to forward to
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	HostPort int32 `json:"hostPort,omitempty"`
}

type ForwardService struct {
	// Name of the service resource to forward to
	// IP and port of the service will be used as the forwarding target
	// Only ClusterIP and LoadBalancer services are supported
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +required
	Name string `json:"name,omitempty"`

	// Namespace of the service resource to forward to
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^[a-z]([-a-z0-9]*[a-z0-9])?$`
	// +optional
	Namespace *string `json:"namespace,omitempty"`

	// Port of the service resource to forward to
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Port *int32 `json:"port,omitempty"`
}

type Forward struct {
	// Scheme is the scheme to use for the forwarding, (http or https)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Enum=http;https
	// +required
	Scheme string `json:"scheme,omitempty"`

	// Service resource reference to forward to
	// This is the preferred way to forward to a service than the host configuration
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	Service *ForwardService `json:"service,omitempty"`

	// Host configuration, the Service configuration is the preferred way
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	Host *ForwardHost `json:"host,omitempty"`

	// Add a path for sub-folder forwarding
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^\/([a-zA-Z0-9._~-]+\/?)*$`
	// +optional
	Path string `json:"path,omitempty"`

	// AdvancedConfig is the advanced configuration for the proxyhost, at your own risk
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +optional
	AdvancedConfig string `json:"advancedConfig,omitempty"`
}

type CustomLocation struct {
	// Define location Location path
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^\/([a-zA-Z0-9._~-]+\/?)*$`
	// +required
	LocationPath string `json:"locationPath,omitempty"`

	// The Service forward configuration for the custom location
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=object
	// +required
	Forward Forward `json:"forward,omitempty"`
}

type ProxyHostAccessList struct {
	// If you know ID of an access list, you can put it here
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=integer
	// +optional
	RemoteId *int `json:"remoteId,omitempty"`

	// The access list resource name
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +optional
	Name string `json:"name,omitempty"`

	// Namespace of the access list resource
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^[a-z]([-a-z0-9]*[a-z0-9])?$`
	// +optional
	Namespace *string `json:"namespace,omitempty"`
}

// ProxyHostSpec defines the desired state of ProxyHost
type ProxyHostSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Token resource, if not provided, the operator will try to find a token with `token-nginxpm` name in the same namespace as the proxyhost is created or in the `nginxpm-operator-system` namespace or in the `default` namespace
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	Token *TokenName `json:"token,omitempty"`

	// DomainNames is the list of domain names to add to the proxyhost
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=10
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=array
	// +required
	DomainNames []DomainName `json:"domainNames,omitempty"`

	// If set to true (the default), it will bind and update an existing remote proxy host if the domain names match; otherwise, it will create a new one.
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	BindExisting bool `json:"bindExisting,omitempty"`

	// CachingEnabled is the flag to enable or disable caching, default is false
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	CachingEnabled bool `json:"cachingEnabled,omitempty"`

	// BlockExploits is the flag to enable or disable blocking exploits, default is true
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	BlockExploits bool `json:"blockExploits,omitempty"`

	// WebsocketSupport is the flag to enable or disable websocket support, default is true
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	WebsocketSupport bool `json:"websocketSupport,omitempty"`

	// AccessList to add to the proxyhost
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	AccessList *ProxyHostAccessList `json:"accessList,omitempty"`

	// Ssl configuration for the proxyhost, default is autoCertificateRequest:true
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	Ssl *Ssl `json:"ssl,omitempty"`

	// The Service forward configuration for the proxyhost
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=object
	// +required
	Forward Forward `json:"forward,omitempty"`

	// CustomLocations is the list of custom locations to add to the proxyhost
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=array
	// +optional
	CustomLocations []CustomLocation `json:"customLocations,omitempty"`
}

// ProxyHostStatus defines the observed state of ProxyHost
type ProxyHostStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ProxyHost ID in the Nginx Proxy Manager instance
	Id *int `json:"id,omitempty"`

	// ProxyHost certificate ID in the Nginx Proxy Manager instance
	CertificateId *int `json:"certificateId,omitempty"`

	// Whether the ProxyHost was bound with an existing proxyhost
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	Bound bool `json:"bound,omitempty"`

	// Represents the observations of a ProxyHost's current state.
	// ProxyHost.status.conditions.type are: "Available", "Progressing", and "Degraded"
	// ProxyHost.status.conditions.status are one of True, False, Unknown.
	// ProxyHost.status.conditions.reason the value should be a CamelCase string and producers of specific
	// condition types may define expected values and meanings for this field, and whether the values
	// are considered a guaranteed API.
	// ProxyHost.status.conditions.Message is a human readable message indicating details about the transition.
	// For further information see: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ID",type="integer",JSONPath=".status.id"
// +kubebuilder:printcolumn:name="CertificateId",type="string",JSONPath=".status.certificateId"
// +kubebuilder:printcolumn:name="DomainNames",type="string",JSONPath=".spec.domainNames"
// +kubebuilder:printcolumn:name="Bound",type="boolean",JSONPath=".status.bound"

// ProxyHost is the Schema for the proxyhosts API
type ProxyHost struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProxyHostSpec   `json:"spec,omitempty"`
	Status ProxyHostStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProxyHostList contains a list of ProxyHost
type ProxyHostList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProxyHost `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProxyHost{}, &ProxyHostList{})
}
