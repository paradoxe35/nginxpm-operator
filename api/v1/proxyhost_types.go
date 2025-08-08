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

// +kubebuilder:validation:Required
// +kubebuilder:validation:Type=string
// +kubebuilder:validation:Pattern=`^(\*\.)?[a-zA-Z0-9-]+(\.[a-zA-Z0-9-]+)*\.[a-zA-Z]{2,}$`
// +required
type DomainName string

// +kubebuilder:validation:Required
// +kubebuilder:validation:Type=string
// +kubebuilder:validation:Pattern=`^(\*\.)?[a-zA-Z0-9-]+(\.[a-zA-Z0-9-]+)*\.[a-zA-Z]{2,}(:[0-9]{1,5})?$`
// +required
type DomainNameWithPort string

type SslCustomCertificate struct {
	// Name specifies the CustomCertificate resource to use for SSL/TLS.
	// The referenced CustomCertificate must exist and contain valid certificate data.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +required
	Name string `json:"name,omitempty"`

	// Namespace of the CustomCertificate resource.
	// If not specified, uses the same namespace as the ProxyHost.
	// Must follow Kubernetes namespace naming conventions.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^[a-z]([-a-z0-9]*[a-z0-9])?$`
	// +optional
	Namespace *string `json:"namespace,omitempty"`
}

type SslLetsEncryptCertificate struct {
	// Name specifies the LetsEncryptCertificate resource to use for SSL/TLS.
	// The referenced certificate must exist and be valid for the proxy domains.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +required
	Name string `json:"name,omitempty"`

	// Namespace of the LetsEncryptCertificate resource.
	// If not specified, uses the same namespace as the ProxyHost.
	// Must follow Kubernetes namespace naming conventions.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^[a-z]([-a-z0-9]*[a-z0-9])?$`
	// +optional
	Namespace *string `json:"namespace,omitempty"`
}

type ProxyHostSsl struct {
	// AutoCertificateRequest enables automatic Let's Encrypt certificate provisioning.
	// When true, NPM will automatically request and manage certificates for the domains.
	// Requires valid domain ownership and accessibility for HTTP-01 challenge.
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	AutoCertificateRequest bool `json:"autoCertificateRequest,omitempty"`

	// LetsEncryptCertificate references a managed Let's Encrypt certificate resource.
	// Takes precedence over AutoCertificateRequest when specified.
	// The certificate must be valid for all domains in this ProxyHost.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	LetsEncryptCertificate *SslLetsEncryptCertificate `json:"letsEncryptCertificate,omitempty"`

	// CustomCertificate references a managed custom SSL/TLS certificate resource.
	// Takes highest precedence - overrides both LetsEncryptCertificate and AutoCertificateRequest.
	// Use for certificates from commercial CAs or self-signed certificates.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	CustomCertificate *SslCustomCertificate `json:"customCertificate,omitempty"`

	// CertificateId directly references an existing certificate ID in NPM.
	// Highest priority - overrides all other certificate configurations.
	// Use when binding to pre-existing NPM certificates not managed by this operator.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=integer
	// +optional
	CertificateId *int `json:"certificateId,omitempty"`

	// LetsEncryptEmail is the contact email for Let's Encrypt notifications.
	// Required when AutoCertificateRequest is true.
	// Receives certificate expiration and account-related notifications.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	// +optional
	LetsEncryptEmail *string `json:"letsEncryptEmail,omitempty"`

	// SslForced enables automatic HTTP to HTTPS redirection.
	// When true (default), all HTTP requests are redirected to HTTPS.
	// Set to false to allow both HTTP and HTTPS access.
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	SslForced bool `json:"sslForced,omitempty"`

	// Http2Support enables HTTP/2 protocol support for improved performance.
	// HTTP/2 provides multiplexing, server push, and header compression.
	// Default is true. Disable only if clients have compatibility issues.
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	Http2Support bool `json:"http2Support,omitempty"`

	// HstsEnabled activates HTTP Strict Transport Security headers.
	// HSTS forces browsers to use HTTPS and prevents protocol downgrade attacks.
	// Default is false. Enable for enhanced security on production sites.
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	HstsEnabled bool `json:"hstsEnabled,omitempty"`

	// HstsSubdomains extends HSTS policy to all subdomains.
	// When true, includeSubDomains directive is added to HSTS header.
	// Only effective when HstsEnabled is true. Use with caution on shared domains.
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	HstsSubdomains bool `json:"hstsSubdomains,omitempty"`
}

type ForwardHost struct {
	// HostName specifies the target host for forwarding requests.
	// Accepts valid DNS names (e.g., "backend.local") or IP addresses (IPv4/IPv6).
	// This host receives the proxied traffic from Nginx.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^((([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)\.)*[a-zA-Z]{2,63}|((25[0-5]|2[0-4][0-9]|[0-1]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[0-1]?[0-9][0-9]?)|(([0-9a-fA-F]{1,4}:){7}([0-9a-fA-F]{1,4}|:))|(::([0-9a-fA-F]{1,4}:){0,6}[0-9a-fA-F]{1,4}))$`
	// +required
	HostName string `json:"hostName,omitempty"`

	// HostPort specifies the TCP port on the target host.
	// Must be a valid port number (1-65535).
	// Common values: 80 (HTTP), 443 (HTTPS), 8080 (alternative HTTP).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +required
	HostPort int32 `json:"hostPort,omitempty"`
}

type ForwardService struct {
	// Name of the Kubernetes Service to forward requests to.
	// The Service's ClusterIP and port will be used as the forwarding target.
	// Supports ClusterIP and LoadBalancer service types.
	// Service must be accessible from the Nginx Proxy Manager instance.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +required
	Name string `json:"name,omitempty"`

	// Namespace of the target Kubernetes Service.
	// If not specified, uses the same namespace as the ProxyHost.
	// Must follow Kubernetes namespace naming conventions.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^[a-z]([-a-z0-9]*[a-z0-9])?$`
	// +optional
	Namespace *string `json:"namespace,omitempty"`

	// Port overrides the Service port selection.
	// If not specified, uses the first port from the Service definition.
	// Use when the Service exposes multiple ports and you need a specific one.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Port *int32 `json:"port,omitempty"`
}

type ProxyHostForward struct {
	// Scheme defines the protocol for upstream communication.
	// "http" for unencrypted traffic, "https" for TLS-encrypted traffic.
	// Choose based on your backend service's configuration.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Enum=http;https
	// +required
	Scheme string `json:"scheme,omitempty"`

	// Service references a Kubernetes Service as the forwarding target.
	// Mutually exclusive with Hosts field.
	// The operator will resolve the Service to its ClusterIP automatically.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	Service *ForwardService `json:"service,omitempty"`

	// Hosts defines explicit forwarding targets by hostname/IP and port.
	// Takes priority over Service field when both are specified.
	// Use for non-Kubernetes backends or specific host requirements.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=array
	// +optional
	Hosts []ForwardHost `json:"hosts,omitempty"`

	// Path adds a URL path prefix to forwarded requests.
	// Example: "/api" forwards "example.com/users" to "backend/api/users".
	// Must start with "/". Leave empty to forward without path modification.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^\/([a-zA-Z0-9._~-]+\/?)*$`
	// +optional
	Path string `json:"path,omitempty"`

	// AdvancedConfig contains raw Nginx configuration directives.
	// Injected directly into the location block. Use with caution.
	// May override operator-managed settings. For advanced users only.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +optional
	AdvancedConfig string `json:"advancedConfig,omitempty"`
}

type CustomLocation struct {
	// LocationPath defines the URL path pattern for this custom location.
	// Must start with "/". Supports exact matches and prefix matches.
	// Example: "/api" matches all requests starting with "/api".
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^\/([a-zA-Z0-9._~-]+\/?)*$`
	// +required
	LocationPath string `json:"locationPath,omitempty"`

	// Forward specifies the upstream configuration for this location.
	// Overrides the default forward configuration for this specific path.
	// Allows different backends for different URL paths on the same domain.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=object
	// +required
	Forward ProxyHostForward `json:"forward,omitempty"`
}

type ProxyHostAccessList struct {
	// AccessListId directly references an existing NPM access list by ID.
	// Takes precedence over Name field.
	// Use when binding to pre-existing access lists not managed by this operator.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=integer
	// +optional
	AccessListId *int `json:"accessListId,omitempty"`

	// Name specifies the AccessList resource to apply to this ProxyHost.
	// The AccessList defines authentication and IP-based access controls.
	// Must reference an existing AccessList in the cluster.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +optional
	Name string `json:"name,omitempty"`

	// Namespace of the AccessList resource.
	// If not specified, uses the same namespace as the ProxyHost.
	// Must follow Kubernetes namespace naming conventions.
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

	// Token references the authentication token for the Nginx Proxy Manager API.
	// If not provided, the operator will search for a token named "token-nginxpm" in:
	// 1. The same namespace as this ProxyHost
	// 2. The "nginxpm-operator-system" namespace
	// 3. The "default" namespace
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	Token *TokenName `json:"token,omitempty"`

	// DomainNames lists the domains this proxy will handle.
	// Supports standard domains ("example.com"), wildcards ("*.example.com"), and ports (e.g., "example.com:8080").
	// All domains must point to the Nginx Proxy Manager instance.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=10
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=array
	// +required
	DomainNames []DomainNameWithPort `json:"domainNames,omitempty"`

	// BindExisting controls the operator's behavior with existing NPM proxy hosts.
	// When true (default): Updates existing proxy hosts with matching domains.
	// When false: Always creates new proxy hosts, may cause conflicts.
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	BindExisting bool `json:"bindExisting,omitempty"`

	// CachingEnabled activates Nginx caching for improved performance.
	// When true, static content and responses are cached according to cache headers.
	// Default is false. Enable for better performance with cacheable content.
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	CachingEnabled bool `json:"cachingEnabled,omitempty"`

	// BlockExploits enables common exploit protection rules.
	// Blocks various SQL injection, XSS, and other common web exploits.
	// Default is true. Only disable if it causes issues with legitimate traffic.
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	BlockExploits bool `json:"blockExploits,omitempty"`

	// WebsocketSupport enables WebSocket protocol proxying.
	// Required for real-time applications using WebSocket connections.
	// Default is true. Disable only if WebSocket is not needed.
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	WebsocketSupport bool `json:"websocketSupport,omitempty"`

	// AccessList configures authentication and access control for this proxy.
	// References an AccessList resource defining auth requirements and IP restrictions.
	// Leave empty for public access without authentication.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	AccessList *ProxyHostAccessList `json:"accessList,omitempty"`

	// Ssl configures SSL/TLS settings for this proxy host.
	// Controls certificate management, HTTPS redirection, and security headers.
	// If not specified, defaults to automatic Let's Encrypt certificate.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	Ssl *ProxyHostSsl `json:"ssl,omitempty"`

	// Forward defines the default upstream configuration for all requests.
	// Specifies where and how to forward incoming traffic.
	// Can be overridden for specific paths using CustomLocations.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=object
	// +required
	Forward ProxyHostForward `json:"forward,omitempty"`

	// CustomLocations defines path-specific forwarding rules.
	// Each location can have different upstream servers and configurations.
	// Useful for routing different paths to different backend services.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=array
	// +optional
	CustomLocations []CustomLocation `json:"customLocations,omitempty"`
}

// ProxyHostStatus defines the observed state of ProxyHost
type ProxyHostStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Id represents the unique identifier assigned by the Nginx Proxy Manager instance.
	// This field is populated after successful creation/synchronization with NPM.
	Id *int `json:"id,omitempty"`

	// CertificateId indicates the SSL certificate ID currently used by this proxy.
	// References the certificate in NPM that handles HTTPS for this proxy.
	// Updated when certificates are changed or renewed.
	CertificateId *int `json:"certificateId,omitempty"`

	// Bound indicates if this resource was linked to an existing NPM proxy host.
	// When true, the operator found and adopted an existing proxy with matching domains.
	// When false, a new proxy host was created in NPM.
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	Bound bool `json:"bound,omitempty"`

	// Online reflects the proxy host's operational status in NPM.
	// True indicates the proxy is active and serving traffic.
	// False may indicate configuration errors or NPM issues.
	// +kubebuilder:validation:Enum=true;false
	// +kubebuilder:validation:Default=false
	Online bool `json:"online,omitempty"`

	// Conditions represent the current state of the ProxyHost resource.
	// Common condition types:
	// - "Ready": ProxyHost is configured and serving traffic
	// - "Progressing": Configuration changes are being applied
	// - "Degraded": ProxyHost has issues but may partially function
	// - "CertificateReady": SSL certificate is valid and active
	// Each condition includes status (True/False/Unknown), reason, and message fields.

	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ID",type="integer",JSONPath=".status.id"
// +kubebuilder:printcolumn:name="Online",type="boolean",JSONPath=".status.online"
// +kubebuilder:printcolumn:name="CertificateId",type="string",JSONPath=".status.certificateId"
// +kubebuilder:printcolumn:name="Domains",type="string",JSONPath=".spec.domainNames"
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
