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
	// TypeDegradedPaasNs represents the status used when the PaasNs is deleted and the finalizer operations are yet to occur.
	TypeDegradedPaasNs = "Degraded"
)

// PaasNSSpec defines the desired state of PaasNS
type PaasNSSpec struct {
	Paas       string            `json:"paas"`
	Groups     []string          `json:"groups,omitempty"`
	SshSecrets map[string]string `json:"sshSecrets,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=paasns,scope=Namespaced

// PaasNS is the Schema for the paasns API
type PaasNS struct {
	metav1.TypeMeta   `json:",inline"`
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

func (p PaasNS) ClonedLabels() map[string]string {
	labels := make(map[string]string)
	for key, value := range p.Labels {
		if key != "app.kubernetes.io/instance" {
			labels[key] = value
		}
	}
	return labels
}

func (p PaasNS) IsItMe(reference metav1.OwnerReference) bool {
	if p.APIVersion != reference.APIVersion {
		return false
	} else if p.Kind != reference.Kind {
		return false
	} else if p.Name != reference.Name {
		return false
	}
	return true
}

func (p PaasNS) AmIOwner(references []metav1.OwnerReference) bool {
	for _, reference := range references {
		if p.IsItMe(reference) {
			return true
		}
	}
	return false
}

//+kubebuilder:object:root=true

// PaasNSList contains a list of PaasNS
type PaasNSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PaasNS `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PaasNS{}, &PaasNSList{})
}

// PaasStatus defines the observed state of Paas
type PaasNsStatus struct {
	// Deprecated: use paasns.status.conditions instead
	Messages   []string           `json:"messages,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// Deprecated: use paasns.status.conditions instead
func (ps *PaasNsStatus) Truncate() {
	ps.Messages = []string{}
}

// Deprecated: use paasns.status.conditions instead
func (ps *PaasNsStatus) GetMessages() []string {
	return ps.Messages
}
