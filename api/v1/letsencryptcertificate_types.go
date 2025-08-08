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

type DnsChallengeProviderCredentialsSecret struct {
	// Name specifies the Kubernetes Secret containing DNS provider credentials.
	// This Secret must contain the required authentication fields for the DNS provider.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +required
	Name string `json:"name,omitempty"`
}

type DnsChallengeProviderCredentials struct {
	// Secret references the Kubernetes Secret containing DNS challenge provider credentials.
	// The Secret structure depends on the DNS provider being used (e.g., API keys, tokens).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=object
	// +required
	Secret DnsChallengeProviderCredentialsSecret `json:"secret,omitempty"`
}

type DnsChallenge struct {
	// Provider specifies the DNS provider for ACME DNS-01 challenge validation.
	// Supported providers include major DNS services like Cloudflare, Route53, Azure, etc.
	// The provider determines which credentials are required in the Secret.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Enum=acmedns;aliyun;azure;bunny;cloudflare;cloudns;cloudxns;constellix;corenetworks;cpanel;desec;duckdns;digitalocean;directadmin;dnsimple;dnsmadeeasy;dnsmulti;dnspod;domainoffensive;domeneshop;dynu;easydns;eurodns;freedns;gandi;godaddy;google;googledomains;he;hetzner;infomaniak;inwx;ionos;ispconfig;isset;joker;linode;loopia;luadns;namecheap;netcup;njalla;nsone;oci;ovh;plesk;porkbun;powerdns;regru;rfc2136;route53;strato;timeweb;transip;tencentcloud;vultr;websupport
	// +required
	Provider string `json:"provider,omitempty"`

	// ProviderCredentials references the Secret containing authentication for the DNS provider.
	// Required fields in the Secret vary by provider (e.g., CF_Token for Cloudflare).
	// These credentials must have permissions to create DNS TXT records for validation.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:XPreserveUnknownFields
	// +required
	ProviderCredentials DnsChallengeProviderCredentials `json:"providerCredentials,omitempty"`

	// PropagationSeconds defines the wait time after DNS record creation before validation.
	// This allows DNS changes to propagate across name servers.
	// Default is 0, which uses the provider's default propagation time.
	// +kubebuilder:default:=0
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=integer
	// +optional
	PropagationSeconds int16 `json:"propagationSeconds,omitempty"`
}

// LetsEncryptCertificateSpec defines the desired state of LetsEncryptCertificate
type LetsEncryptCertificateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Token references the authentication token for the Nginx Proxy Manager API.
	// If not provided, the operator will search for a token named "token-nginxpm" in:
	// 1. The same namespace as this LetsEncryptCertificate
	// 2. The "nginxpm-operator-system" namespace
	// 3. The "default" namespace
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	Token *TokenName `json:"token,omitempty"`

	// DomainNames lists the domain names to include in the Let's Encrypt certificate.
	// Supports wildcards (e.g., "*.example.com") and multiple domains.
	// All domains must be under your control for validation to succeed.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=10
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=array
	// +required
	DomainNames []DomainName `json:"domainNames,omitempty"`

	// LetsEncryptEmail is the contact email for Let's Encrypt account and notifications.
	// This email receives important notices about certificate expiration and account issues.
	// Must be a valid email address format.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Format:=email
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	// +required
	LetsEncryptEmail string `json:"letsEncryptEmail,omitempty"`

	// DnsChallenge configures DNS-01 challenge for domain validation.
	// Required for wildcard certificates or when HTTP-01 challenge is not feasible.
	// When not specified, HTTP-01 challenge is used by default.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	DnsChallenge *DnsChallenge `json:"dnsChallenge,omitempty"`
}

// LetsEncryptCertificateStatus defines the observed state of LetsEncryptCertificate
type LetsEncryptCertificateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Id represents the unique identifier assigned by the Nginx Proxy Manager instance.
	// This field is populated after successful certificate creation in NPM.
	Id *int `json:"id,omitempty"`

	// Bound indicates if this resource was linked to an existing NPM certificate.
	// When true, the operator found and adopted an existing certificate with matching domains.
	// When false, a new certificate was created in NPM.
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	Bound bool `json:"bound,omitempty"`

	// DomainNames contains the actual domains in the issued certificate.
	// This is preserved in status because the certificate remains valid for these domains
	// even if spec.domainNames is modified (which would trigger a new certificate).
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=10
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=array
	// +required
	DomainNames []string `json:"domainNames,omitempty"`

	// ExpiresOn indicates when the Let's Encrypt certificate will expire.
	// Format: ISO 8601 date-time string.
	// Let's Encrypt certificates are valid for 90 days and should be renewed before expiration.
	// +optional
	ExpiresOn *string `json:"expiresOn,omitempty"`

	// Conditions represent the current state of the LetsEncryptCertificate resource.
	// Common condition types include "Ready", "Issued", "Renewing", and "ValidationFailed".
	// The "Ready" condition indicates if the certificate is successfully issued and active.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ID",type="integer",JSONPath=".status.id"
// +kubebuilder:printcolumn:name="DomainNames",type="string",JSONPath=".spec.domainNames"
// +kubebuilder:printcolumn:name="Bound",type="boolean",JSONPath=".status.bound"
// +kubebuilder:printcolumn:name="ExpiresOn",type="string",JSONPath=".status.expiresOn"

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
