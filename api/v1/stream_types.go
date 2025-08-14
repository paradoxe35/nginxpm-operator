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

type StreamForward struct {
	// TCPForwarding enables TCP protocol forwarding for this stream.
	// When true, TCP traffic on the incoming port is forwarded to the target.
	// Default is true. Set to false if only UDP forwarding is needed.
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	TCPForwarding bool `json:"tcpForwarding,omitempty"`

	// UDPForwarding enables UDP protocol forwarding for this stream.
	// When true, UDP traffic on the incoming port is forwarded to the target.
	// Default is true. Set to false if only TCP forwarding is needed.
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	UDPForwarding bool `json:"udpForwarding,omitempty"`

	// Service references a Kubernetes Service as the forwarding target.
	// The Service's ClusterIP and port will be used for stream forwarding.
	// Mutually exclusive with Hosts field.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	Service *ForwardService `json:"service,omitempty"`

	// Hosts defines explicit forwarding targets by hostname/IP and port.
	// Takes priority over Service field when both are specified.
	// Use for non-Kubernetes backends or load balancing across multiple hosts.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=array
	// +optional
	Hosts []ForwardHost `json:"hosts,omitempty"`
}

type StreamSsl struct {
	// LetsEncryptCertificate references a managed Let's Encrypt certificate resource.
	// Enables TLS/SSL for the stream using Let's Encrypt certificates.
	// The certificate must be valid for stream usage (not just HTTP/HTTPS).
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	LetsEncryptCertificate *SslLetsEncryptCertificate `json:"letsEncryptCertificate,omitempty"`

	// CustomCertificate references a managed custom SSL/TLS certificate resource.
	// Takes priority over LetsEncryptCertificate when both are specified.
	// Use for certificates from commercial CAs or self-signed certificates.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	CustomCertificate *SslCustomCertificate `json:"customCertificate,omitempty"`

	// CertificateId directly references an existing certificate ID in NPM.
	// Highest priority - overrides both CustomCertificate and LetsEncryptCertificate.
	// Use when binding to pre-existing NPM certificates not managed by this operator.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=integer
	// +optional
	CertificateId *int `json:"certificateId,omitempty"`
}

// StreamSpec defines the desired state of Stream.
type StreamSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Token references the authentication token for the Nginx Proxy Manager API.
	// If not provided, the operator will search for a token named "token-nginxpm" in:
	// 1. The same namespace as this Stream
	// 2. The "nginxpm-operator-system" namespace
	// 3. The "default" namespace
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	Token *TokenName `json:"token,omitempty"`

	// IncomingPort defines the port where the stream will listen for connections.
	// Must be available and not in use by other services.
	// Common ranges: 1024-65535 for non-privileged ports.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +required
	IncomingPort int `json:"incomingPort,omitempty"`

	// OverwriteIncomingPortWithForwardPort allows automatic port matching.
	// When true, the incoming port is set to match the forwarding port.
	// Useful for transparent proxying where ports should remain the same.
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	OverwriteIncomingPortWithForwardPort bool `json:"overwriteIncomingPortWithForwardPort,omitempty"`

	// Forward defines the upstream configuration for this stream.
	// Specifies protocol support (TCP/UDP) and target hosts or services.
	// At least one protocol (TCP or UDP) must be enabled.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=object
	// +required
	Forward StreamForward `json:"forward,omitempty"`

	// Ssl configures TLS/SSL termination for this stream.
	// Enables encrypted connections for TCP streams (not applicable to UDP).
	// Leave empty for unencrypted stream forwarding.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	Ssl *StreamSsl `json:"ssl,omitempty"`
}

// StreamStatus defines the observed state of Stream.
type StreamStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Id represents the unique identifier assigned by the Nginx Proxy Manager instance.
	// This field is populated after successful stream creation in NPM.
	Id *int `json:"id,omitempty"`

	// IncomingPort shows the actual listening port configured in NPM.
	// May differ from spec if port conflicts were resolved.
	// This is the port clients should connect to.
	IncomingPort *int `json:"incomingPort,omitempty"`

	// ForwardingPort shows the target port being forwarded to.
	// Reflects the actual upstream port configuration in NPM.
	// Useful for debugging connection issues.
	ForwardingPort *int `json:"forwardingPort,omitempty"`

	// Online reflects the stream's operational status in NPM.
	// True indicates the stream is active and forwarding traffic.
	// False may indicate configuration errors or port conflicts.
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=boolean
	// +optional
	Online bool `json:"online,omitempty"`

	// Conditions represent the current state of the Stream resource.
	// Common condition types include "Ready", "PortAvailable", and "Synced".
	// The "Ready" condition indicates if the stream is successfully configured and active.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ID",type="integer",JSONPath=".status.id"
// +kubebuilder:printcolumn:name="Online",type="boolean",JSONPath=".status.online"
// +kubebuilder:printcolumn:name="Incoming",type="integer",JSONPath=".status.incomingPort"
// +kubebuilder:printcolumn:name="Forwarding",type="integer",JSONPath=".status.forwardingPort"
// +kubebuilder:printcolumn:name="TCP",type="boolean",JSONPath=".spec.forward.tcpForwarding"
// +kubebuilder:printcolumn:name="UDP",type="boolean",JSONPath=".spec.forward.udpForwarding"

// Stream is the Schema for the streams API.
type Stream struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StreamSpec   `json:"spec,omitempty"`
	Status StreamStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StreamList contains a list of Stream.
type StreamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Stream `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Stream{}, &StreamList{})
}
