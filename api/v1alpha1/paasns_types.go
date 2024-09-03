/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PaasNSSpec defines the desired state of PaasNS
type PaasNSSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of PaasNS. Edit paasns_types.go to remove/update
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
	// Important: Run "make" to regenerate code after modifying this file
	Messages []string `json:"messages,omitempty"`
}

func (ps *PaasNsStatus) Truncate() {
	ps.Messages = []string{}
}

func (ps *PaasNsStatus) AddMessage(level PaasStatusLevel, action PaasStatusAction, obj client.Object, message string) {
	namespacedName := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
	ps.Messages = append(ps.Messages,
		fmt.Sprintf("%s: %s for %s (%s) %s", level, action, namespacedName.String(), obj.GetObjectKind().GroupVersionKind().String(), message),
	)
}

func (ps *PaasNsStatus) GetMessages() []string {
	return ps.Messages
}

func (ps *PaasNsStatus) AddMessages(msgs []string) {
	ps.Messages = append(ps.Messages, msgs...)
}
