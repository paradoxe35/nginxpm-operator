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
	// Has TCP Forwarding
	// +kubebuilder:validation:Enum=true;false
	// +kubebuilder:validation:Default=true
	TCPForwarding bool `json:"tcpForwarding,omitempty"`

	// Has UDP Forwarding
	// +kubebuilder:validation:Enum=true;false
	// +kubebuilder:validation:Default=true
	UDPForwarding bool `json:"udpForwarding,omitempty"`

	// Service resource reference to be forwarded to
	// This is the preferred method to forward to a service rather than using host configuration.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	Service *ForwardService `json:"service,omitempty"`

	// Configure your host forwarding settings here; using the Service configuration is recommended.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	Host *ForwardHost `json:"host,omitempty"`
}

type StreamSsl struct {
	// Letsencrypt Certificate name managed by the letsencryptCertificate resource
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	LetsEncryptCertificate *SslLetsEncryptCertificate `json:"letsEncryptCertificate,omitempty"`

	// Custom Certificate name managed by the customCertificate resource
	// CustomCertificate has priority over LetsencryptCertificate
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	CustomCertificate *SslCustomCertificate `json:"customCertificate,omitempty"`

	// Bind existing certificate id to the stream
	// CustomCertificate has priority over LetsencryptCertificate and  CustomCertificate
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=integer
	// +optional
	CertificateId *int `json:"certificateId,omitempty"`
}

// StreamSpec defines the desired state of Stream.
type StreamSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Token resource, if not provided, the operator will try to find a token with `token-nginxpm` name in the same namespace as the proxyhost is created or in the `nginxpm-operator-system` namespace or in the `default` namespace
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	Token *TokenName `json:"token,omitempty"`

	// Incoming Port
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +required
	IncomingPort int `json:"incomingPort,omitempty"`

	// If True the incoming port will be overwritten with the forward port
	// +kubebuilder:validation:Enum=true;false
	// +kubebuilder:validation:Default=false
	OverwriteIncomingPortWithForwardPort bool `json:"overwriteIncomingPortWithForwardPort,omitempty"`

	// Stream forward configuration
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=object
	// +required
	Forward StreamForward `json:"forward,omitempty"`

	// Ssl configuration for the stream
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=object
	// +optional
	Ssl *StreamSsl `json:"ssl,omitempty"`
}

// StreamStatus defines the observed state of Stream.
type StreamStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Stream ID from remote Nginx Proxy Manager instance
	Id *int `json:"id,omitempty"`

	// Incoming port
	IncomingPort *int `json:"incomingPort,omitempty"`

	// Forwarding port
	ForwardingPort *int `json:"forwardingPort,omitempty"`

	// Online status from remote Nginx Proxy Manager instance
	// +kubebuilder:validation:Enum=true;false
	// +kubebuilder:validation:Default=false
	Online bool `json:"online,omitempty"`

	// Represents the observations of a Stream's current state.
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
