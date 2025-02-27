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

// StreamSpec defines the desired state of Stream.
type StreamSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Incoming Port
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +required
	IncomingPort int `json:"incomingPort,omitempty"`
}

// StreamStatus defines the observed state of Stream.
type StreamStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Stream ID from remote  Nginx Proxy Manager instance
	Id *int `json:"id,omitempty"`

	// Online status from remote  Nginx Proxy Manager instance
	Online bool `json:"online,omitempty"`

	// Represents the observations of a Stream's current state.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ID",type="integer",JSONPath=".status.id"
// +kubebuilder:printcolumn:name="Online",type="boolean",JSONPath=".status.online"
// +kubebuilder:printcolumn:name="Incoming Port",type="boolean",JSONPath=".spec.incomingPort"

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
