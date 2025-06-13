/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha2

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/belastingdienst/opr-paas/v2/internal/fields"
	"github.com/belastingdienst/opr-paas/v2/internal/groups"
	paasquota "github.com/belastingdienst/opr-paas/v2/internal/quota"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Definitions to manage status conditions
const (
	// TypeReadyPaas represents the status of the Paas reconciliation
	TypeReadyPaas = "Ready"
	// TypeHasErrorsPaas represents the status used when the Paas reconciliation holds errors.
	TypeHasErrorsPaas = "HasErrors"
	// TypeDegradedPaas represents the status used when the Paas is deleted and the finalizer operations are yet to
	// occur.
	TypeDegradedPaas = "Degraded"
)

// PaasSpec defines the desired state of Paas
type PaasSpec struct {
	// Requestor is an informational field which decides on the requestor (also application responsible)
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Requestor string `json:"requestor"`

	// Quota defines the quotas which should be set on the cluster resource quota as used by this Paas project
	// +kubebuilder:validation:Required
	Quota paasquota.Quota `json:"quota"`

	// Capabilities is a subset of capabilities that will be available in this Paas Project
	// +kubebuilder:validation:Optional
	Capabilities PaasCapabilities `json:"capabilities"`

	// Groups define k8s groups, based on an LDAP query or a list of LDAP users, which get access to the namespaces
	// belonging to this Paas. Per group, RBAC roles can be defined.
	// +kubebuilder:validation:Optional
	Groups PaasGroups `json:"groups"`

	// Namespaces can be used to define extra namespaces to be created as part of this Paas project
	// +kubebuilder:validation:Optional
	Namespaces PaasNamespaces `json:"namespaces"`

	// Secrets must be encrypted with a public key, for which the private key should be added to the DecryptKeySecret
	// +kubebuilder:validation:Optional
	Secrets map[string]string `json:"secrets"`

	// Indicated by which 3rd party Paas this Paas is managed
	// +kubebuilder:validation:Optional
	ManagedByPaas string `json:"managedByPaas"`
}

// PaasCapability holds all information for a capability
type PaasCapability struct {
	// Custom fields to configure this specific Capability
	// +kubebuilder:validation:Optional
	CustomFields map[string]string `json:"custom_fields"`
	// This project has its own ClusterResourceQuota settings
	// +kubebuilder:validation:Optional
	Quota paasquota.Quota `json:"quota"`
	// Secrets must be encrypted with a public key, for which the private key should be added to the DecryptKeySecret
	// +kubebuilder:validation:Optional
	Secrets map[string]string `json:"secrets"`
	// You can enable extra permissions for the service accounts belonging to this capability
	// Exact definitions is configured in Paas Configmap
	// +kubebuilder:validation:Optional
	ExtraPermissions bool `json:"extra_permissions"`
}

// Quotas returns the quota for this capability
func (pc PaasCapability) Quotas() (pq paasquota.Quota) {
	return pc.Quota
}

// CapExtraFields returns all extra fields that are configured for a capability
func (pc *PaasCapability) CapExtraFields(
	fieldConfig map[string]ConfigCustomField,
) (elements fields.Elements, err error) {
	elements = make(fields.Elements)
	var issues []error
	for key, value := range pc.CustomFields {
		if _, exists := fieldConfig[key]; !exists {
			issues = append(issues, fmt.Errorf("custom field %s is not configured in capability config", key))
		} else {
			elements[key] = value
		}
	}
	for key, fieldConf := range fieldConfig {
		if value, err := elements.TryGetElementAsString(key); err == nil {
			if matched, err := regexp.Match(fieldConf.Validation, []byte(value)); err != nil {
				issues = append(issues, fmt.Errorf("could not validate value %s: %w", value, err))
			} else if !matched {
				issues = append(issues,
					fmt.Errorf("invalid value %s (does not match %s)", value, fieldConf.Validation),
				)
			}
		} else if fieldConf.Required {
			issues = append(issues, fmt.Errorf("value %s is required", key))
		} else if fieldConf.Template == "" || fieldConf.Default != "" {
			elements[key] = fieldConf.Default
		}
	}
	if len(issues) > 0 {
		return nil, errors.Join(issues...)
	}
	return elements, nil
}

// PaasCapabilities holds all capabilities enabled in a Paas
type PaasCapabilities map[string]PaasCapability

// PaasGroup can hold information about a group in the paas.spec.groups block
type PaasGroup struct {
	// A fully qualified LDAP query which will be used by the Group Sync Operator to sync users to the defined group.
	//
	// When set in combination with `users`, the Group Sync Operator will overwrite the manually assigned users.
	// Therefore, this field is mutually exclusive with `group.users`.
	// +kubebuilder:validation:Optional
	Query string `json:"query"`
	// A list of LDAP users which are added to the defined group.
	//
	// When set in combination with `users`, the Group Sync Operator will overwrite the manually assigned users.
	// Therefore, this field is mutually exclusive with `group.query`.
	// +kubebuilder:validation:Optional
	Users []string `json:"users"`
	// List of roles, as defined in the `PaasConfig` which the users in this group get assigned via a rolebinding.
	// +kubebuilder:validation:Optional
	Roles []string `json:"roles"`
}

// PaasGroups hold all groups in a paas.spec.groups
type PaasGroups map[string]PaasGroup

// PaasNamespaces is a key, value store of all defined Namespaces
type PaasNamespaces map[string]PaasNamespace

// PaasNamespace holds all info regarding a Paas managed Namespace (groups and secrets)
type PaasNamespace struct {
	// Keys of groups which should get access to this namespace. When not set it defaults to all groups listed in
	// `spec.groups`.
	// +kubebuilder:validation:Optional
	Groups []string `json:"groups"`
	// Secrets which should exist in this namespace, the values must be encrypted with a key pair referenced by
	// `spec.decryptKeySecret` from the active PaasConfig.
	// +kubebuilder:validation:Optional
	Secrets map[string]string `json:"secrets"`
}

// PaasStatus defines the observed state of Paas
type PaasStatus struct {
	// +kubebuilder:validation:Optional
	//revive:disable-next-line
	Conditions []metav1.Condition `json:"conditions" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:conversion:hub
// +kubebuilder:resource:path=paas,scope=Cluster

// Paas is the Schema for the paas API
type Paas struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PaasSpec   `json:"spec,omitempty"`
	Status PaasStatus `json:"status,omitempty"`
}

// GetConditions is required to allow a Paas to be used as a withStatus interface in our e2e test framework
func (p *Paas) GetConditions() *[]metav1.Condition {
	return &p.Status.Conditions
}

// GetGeneration is required for Paas to be used as v1alpha1.Resource
func (p Paas) GetGeneration() int64 {
	return p.Generation
}

// +kubebuilder:object:root=true

// PaasList contains a list of Paas
type PaasList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Paas `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Paas{}, &PaasList{})
}

// ClonedLabels returns a map of labels that can be used to clone a Paas
func (p Paas) ClonedLabels() map[string]string {
	labels := make(map[string]string)
	for key, value := range p.Labels {
		if key != "app.kubernetes.io/instance" {
			labels[key] = value
		}
	}
	return labels
}

// AmIOwner returns true if the Paas is listed in the given ownerReferences
func (p Paas) AmIOwner(references []metav1.OwnerReference) bool {
	for _, reference := range references {
		if p.IsItMe(reference) {
			return true
		}
	}
	return false
}

// IsItMe returns true if the given OwnerReference is of Kind "Paas" and has the same Name as the Paas.
func (p Paas) IsItMe(reference metav1.OwnerReference) bool {
	if reference.APIVersion != paasAPIVersion ||
		reference.Kind != "Paas" ||
		reference.Name != p.Name {
		return false
	}

	return true
}

// WithoutMe returns a list of OwnerReferences that do not include a reference to the Paas.
func (p Paas) WithoutMe(references []metav1.OwnerReference) (withoutMe []metav1.OwnerReference) {
	for _, reference := range references {
		if !p.IsItMe(reference) {
			withoutMe = append(withoutMe, reference)
		}
	}
	return withoutMe
}

// GroupKey2GroupName can be used to get the real name of a group from it;s key in paas.spec.groups
// This could be either derived from the query, or the key, or empty string (if it is not a valid key)
func (p Paas) GroupKey2GroupName(groupKey string) string {
	if group, exists := p.Spec.Groups[groupKey]; !exists {
		return ""
	} else if len(group.Query) > 0 {
		return group.name(groupKey)
	}
	return fmt.Sprintf("%s-%s", p.Name, groupKey)
}

func (pg PaasGroup) name(defName string) string {
	name := strings.Split(pg.Query, ",")[0]
	if len(name) == 0 {
		return defName
	} else if strings.Contains(name, "=") {
		return strings.Split(name, "=")[1]
	}
	return name
}

// AsGroups can be used to convert PaasGroups to groups.Groups
func (pgs PaasGroups) AsGroups() groups.Groups {
	newGroups := groups.NewGroups()
	newGroups.AddFromStrings(pgs.LdapQueries())
	return *newGroups
}

// LdapQueries can return a list of all ldap queries (which could technically have duplicates)
func (pgs PaasGroups) LdapQueries() []string {
	var queries []string
	for _, group := range pgs {
		if group.Query != "" {
			queries = append(queries, group.Query)
		}
	}
	return queries
}

// ManagedByPaas can be used to retrieve the Paas that manages all namespaces from this Paas,
// which is either stated in p.spec.managedByPaas, or this paas itself.
func (p Paas) ManagedByPaas() string {
	if p.Spec.ManagedByPaas != "" {
		return p.Spec.ManagedByPaas
	}

	return p.Name
}

// Filtered returns a list of PaasGroups which have a key that is in the list of groups, specified as string.
func (pgs PaasGroups) Filtered(paasGroupNames []string) PaasGroups {
	filtered := make(PaasGroups)
	if len(paasGroupNames) == 0 {
		return pgs
	}
	for _, paasGroupName := range paasGroupNames {
		if paasGroup, exists := pgs[paasGroupName]; exists {
			filtered[paasGroupName] = paasGroup
		}
	}
	return filtered
}

// Roles returns a map of groupKeys with the roles defined within that groupKey
func (pgs PaasGroups) Roles() map[string][]string {
	roles := make(map[string][]string)
	for groupKey, group := range pgs {
		roles[groupKey] = group.Roles
	}
	return roles
}

// Keys can return a list of all keys in paas.spec.groups
func (pgs PaasGroups) Keys() (keys []string) {
	for key := range pgs {
		keys = append(keys, key)
	}
	return keys
}
