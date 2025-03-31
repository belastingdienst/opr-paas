/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PaasSpec defines the desired state of Paas
type PaasSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Paas. Edit paas_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// PaasStatus defines the observed state of Paas
type PaasStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Paas is the Schema for the paas API
type Paas struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PaasSpec   `json:"spec,omitempty"`
	Status PaasStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PaasList contains a list of Paas
type PaasList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Paas `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Paas{}, &PaasList{})
}
