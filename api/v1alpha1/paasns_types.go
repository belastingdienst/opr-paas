/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"fmt"

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
)

// PaasNSSpec defines the desired state of PaasNS
type PaasNSSpec struct {
	// The `metadata.name` of the Paas which created the namespace in which this PaasNS is applied
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Paas string `json:"paas"`
	// Keys of the groups, as defined in the related `paas`, which should get access to
	// the namespace created by this PaasNS. When not set, all groups as defined in the related
	// `paas` get access to the namespace created by this PaasNS.
	// +kubebuilder:validation:Optional
	Groups []string `json:"groups"`
	// SSHSecrets which should exist in the namespace created through this PaasNS,
	// the values are the encrypted secrets through Crypt
	// +kubebuilder:validation:Optional
	SSHSecrets map[string]string `json:"sshSecrets"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=paasns,scope=Namespaced

// PaasNS is the Schema for the PaasNS API
type PaasNS struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PaasNSSpec   `json:"spec,omitempty"`
	Status PaasNsStatus `json:"status,omitempty"`
}

func (pns PaasNS) NamespaceName() string {
	if pns.Spec.Paas == "" || pns.Name == "" {
		panic(fmt.Errorf("invalid paas or paasns name (empty)"))
	}

	return fmt.Sprintf("%s-%s", pns.Spec.Paas, pns.Name)
}

func (pns PaasNS) ClonedLabels() map[string]string {
	labels := make(map[string]string)
	for key, value := range pns.Labels {
		if key != "app.kubernetes.io/instance" {
			labels[key] = value
		}
	}
	return labels
}

func (pns PaasNS) IsItMe(reference metav1.OwnerReference) bool {
	if pns.APIVersion != reference.APIVersion {
		return false
	} else if pns.Kind != reference.Kind {
		return false
	} else if pns.Name != reference.Name {
		return false
	}
	return true
}

func (pns PaasNS) AmIOwner(references []metav1.OwnerReference) bool {
	for _, reference := range references {
		if pns.IsItMe(reference) {
			return true
		}
	}
	return false
}

func (pns PaasNS) GetConditions() []metav1.Condition {
	return pns.Status.Conditions
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

// revive:disable:line-length-limit

// PaasNsStatus defines the observed state of Paas
type PaasNsStatus struct {
	// Deprecated: use paasns.status.conditions instead
	// +kubebuilder:validation:Optional
	Messages []string `json:"messages"`
	// +kubebuilder:validation:Optional
	Conditions []metav1.Condition `json:"conditions" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// revive:enable:line-length-limit

// Deprecated: use paasns.status.conditions instead
func (ps *PaasNsStatus) Truncate() {
	ps.Messages = []string{}
}

// Deprecated: use paasns.status.conditions instead
func (ps *PaasNsStatus) GetMessages() []string {
	return ps.Messages
}
