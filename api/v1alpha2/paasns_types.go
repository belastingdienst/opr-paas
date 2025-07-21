/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Definitions to manage status conditions
const (
	// TypeReadyPaasNs represents the status of the PaasNs reconciliation
	TypeReadyPaasNs = "Ready"
	// TypeHasErrorsPaasNs represents the status used when the PaasNs reconciliation holds errors.
	TypeHasErrorsPaasNs = "HasErrors"
	// TypeDegradedPaasNs represents the status used when the PaasNs is deleted
	// and the finalizer operations are yet to occur.
	TypeDegradedPaasNs = "Degraded"

	instanceLabel = "app.kubernetes.io/instance"
)

// PaasNSSpec defines the desired state of PaasNS
type PaasNSSpec struct {
	// Deprecated: this has no function anymore and will be deleted in the next version.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Optional
	Paas string `json:"paas,omitempty"`
	// Keys of the groups, as defined in the related `paas`, which should get access to
	// the namespace created by this PaasNS. When not set, all groups as defined in the related
	// `paas` get access to the namespace created by this PaasNS.
	// +kubebuilder:validation:Optional
	Groups []string `json:"groups,omitempty"`
	// Secrets which should exist in the namespace created through this PaasNS,
	// the values are the encrypted secrets through Crypt
	// +kubebuilder:validation:Optional
	Secrets map[string]string `json:"secrets,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:conversion:hub
// +kubebuilder:resource:path=paasns,scope=Namespaced

// PaasNS is the Schema for the PaasNS API
type PaasNS struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PaasNSSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// PaasNSList contains a list of PaasNS
type PaasNSList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PaasNS `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PaasNS{}, &PaasNSList{})
}

// ClonedLabels returns a map of labels hat can be cloned to sub resources (namespace).
// Deprecated: ClonedLabels is replaced by go templating
func (pns PaasNS) ClonedLabels() map[string]string {
	labels := make(map[string]string)
	for key, value := range pns.Labels {
		if key != "app.kubernetes.io/instance" {
			labels[key] = value
		}
	}
	return labels
}
