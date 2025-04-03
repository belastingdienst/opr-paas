/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/belastingdienst/opr-paas/internal/fields"
	"github.com/belastingdienst/opr-paas/internal/groups"
	paasquota "github.com/belastingdienst/opr-paas/internal/quota"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Definitions to manage status conditions
const (
	// TypeReadyPaas represents the status of the Paas reconciliation
	TypeReadyPaas = "Ready"
	// TypeHasErrorsPaas represents the status used when the Paas reconciliation holds errors.
	TypeHasErrorsPaas = "HasErrors"
	// revive:disable-next-line
	// TypeDegradedPaas represents the status used when the Paas is deleted
	// and the finalizer operations are yet to occur.
	TypeDegradedPaas = "Degraded"
)

// PaasSpec defines the desired state of Paas
type PaasSpec struct {
	// Capabilities is a subset of capabilities that will be available in this Paas Project
	// +kubebuilder:validation:Optional
	Capabilities PaasCapabilities `json:"capabilities"`

	// Requestor is an informational field which decides on the requestor (also application responsible)
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Requestor string `json:"requestor"`

	// Groups define k8s groups, based on an LDAP query or a list of LDAP users, which get access to the namespaces
	// belonging to this Paas. Per group, RBAC roles can be defined.
	// +kubebuilder:validation:Optional
	Groups PaasGroups `json:"groups"`

	// Quota defines the quotas which should be set on the cluster resource quota as used by this Paas project
	// +kubebuilder:validation:Required
	Quota paasquota.Quota `json:"quota"`

	// Namespaces can be used to define extra namespaces to be created as part of this Paas project
	// As the names are used as the names of PaasNs resources, they must comply to the DNS subdomainname regex
	// See https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-subdomain-names for more info
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:items:Pattern=`^[a-z0-9]([a-z0-9-.]{0,251}[a-z0-9])?$`
	Namespaces []string `json:"namespaces"`
	// You can add ssh keys (which is a type of secret) for ArgoCD to use for access to bitBucket.
	// They must be encrypted with a public key, for which the private key should be added to the DecryptKeySecret
	// +kubebuilder:validation:Optional
	SSHSecrets map[string]string `json:"sshSecrets"`

	// Indicated by which 3rd party Paas's ArgoCD this Paas is managed
	// +kubebuilder:validation:Optional
	ManagedByPaas string `json:"managedByPaas"`
}

func (p Paas) ManagedByPaas() string {
	if p.Spec.ManagedByPaas != "" {
		return p.Spec.ManagedByPaas
	}

	return p.Name
}

func (p Paas) PrefixedBoolMap(m map[string]bool) map[string]bool {
	newMap := make(map[string]bool)
	for name, value := range m {
		newMap[fmt.Sprintf("%s-%s", p.Name, name)] = value
	}
	return newMap
}

func (p Paas) GetNsSSHSecrets(ns string) (secrets map[string]string) {
	secrets = make(map[string]string)
	for key, value := range p.Spec.SSHSecrets {
		secrets[key] = value
	}
	if capability, exists := p.Spec.Capabilities[ns]; exists {
		for key, value := range capability.GetSSHSecrets() {
			secrets[key] = value
		}
	}
	return secrets
}

func (p Paas) enabledCapNamespaces() (ns map[string]bool) {
	ns = make(map[string]bool)
	for name, cap := range p.Spec.Capabilities {
		if cap.IsEnabled() {
			ns[name] = true
		}
	}
	return ns
}

func (p Paas) AllCapNamespaces() (ns map[string]bool) {
	ns = make(map[string]bool)
	for name := range p.Spec.Capabilities {
		ns[name] = true
	}
	return ns
}

func (p Paas) PrefixedAllCapNamespaces() (ns map[string]bool) {
	return p.PrefixedBoolMap(p.AllCapNamespaces())
}

func (p Paas) AllEnabledNamespaces() (ns map[string]bool) {
	ns = p.enabledCapNamespaces()
	for name := range p.extraNamespaces() {
		ns[name] = true
	}
	return ns
}

func (p Paas) PrefixedAllEnabledNamespaces() (ns map[string]bool) {
	return p.PrefixedBoolMap(p.AllEnabledNamespaces())
}

func (p Paas) extraNamespaces() (ns map[string]bool) {
	capNs := p.AllCapNamespaces()
	ns = make(map[string]bool)
	for _, name := range p.Spec.Namespaces {
		if _, isCap := capNs[name]; !isCap {
			ns[name] = true
		}
	}
	return ns
}

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

func (pg PaasGroup) Name(defName string) string {
	if name := strings.Split(pg.Query, ",")[0]; len(name) == 0 {
		return defName
	} else if strings.Contains(name, "=") {
		return strings.Split(name, "=")[1]
	} else {
		return name
	}
}

type PaasGroups map[string]PaasGroup

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

func (pgs PaasGroups) Key2Name(key string) string {
	if group, exists := pgs[key]; !exists {
		return ""
	} else {
		return group.Name(key)
	}
}

func (pgs PaasGroups) Names() (paasGroupNames []string) {
	for name, paasGroup := range pgs {
		paasGroupNames = append(paasGroupNames, paasGroup.Name(name))
	}
	return paasGroupNames
}

func (p Paas) GroupKey2GroupName(groupKey string) string {
	if group, exists := p.Spec.Groups[groupKey]; !exists {
		return ""
	} else if len(group.Query) > 0 {
		return group.Name(groupKey)
	} else {
		return fmt.Sprintf("%s-%s", p.Name, p.Spec.Groups.Key2Name(groupKey))
	}
}

func (p Paas) GroupNames() (groupNames []string) {
	for groupKey := range p.Spec.Groups {
		groupNames = append(groupNames, p.GroupKey2GroupName(groupKey))
	}
	return groupNames
}

func (pgs PaasGroups) LdapQueries() []string {
	var queries []string
	for _, group := range pgs {
		if group.Query != "" {
			queries = append(queries, group.Query)
		}
	}
	return queries
}

// Keys() returns the keys of the PaasGroups
func (pgs PaasGroups) Keys() (keys []string) {
	for key := range pgs {
		keys = append(keys, key)
	}
	return keys
}

func (pgs PaasGroups) AsGroups() groups.Groups {
	newGroups := groups.NewGroups()
	newGroups.AddFromStrings(pgs.LdapQueries())
	return *newGroups
}

type PaasCapabilities map[string]PaasCapability

func (pcs PaasCapabilities) AsPrefixedMap(prefix string) PaasCapabilities {
	if prefix == "" {
		return pcs
	}
	caps := make(PaasCapabilities)
	for name, cap := range pcs {
		caps[fmt.Sprintf("%s-%s", prefix, name)] = cap
	}
	return caps
}

func (pcs PaasCapabilities) IsCap(name string) bool {
	if capability, exists := pcs[name]; !exists || !capability.IsEnabled() {
		return false
	}

	return true
}

func (pcs PaasCapabilities) GetCapability(capabilityName string) (capability PaasCapability, err error) {
	if capability, exists := pcs[capabilityName]; !exists {
		return capability, fmt.Errorf("capability %s does not exist", capabilityName)
	} else {
		return capability, nil
	}
}

func (pcs PaasCapabilities) AddCapSSHSecret(capabilityName string, key string, value string) (err error) {
	if capability, err := pcs.GetCapability(capabilityName); err != nil {
		return err
	} else {
		if capability.SSHSecrets == nil {
			capability.SSHSecrets = map[string]string{key: value}
		} else {
			capability.SSHSecrets[key] = value
		}
		pcs[capabilityName] = capability
	}
	return nil
}

func (pcs PaasCapabilities) ResetCapSSHSecret(capabilityName string) (err error) {
	if capability, err := pcs.GetCapability(capabilityName); err != nil {
		return err
	} else {
		capability.SSHSecrets = nil
		pcs[capabilityName] = capability
	}
	return nil
}

// TODO: Enabled is a leftover from old capability implementation. Remove with new API.

type PaasCapability struct {
	// Do we want to use this capability, default false
	// +kubebuilder:validation:Optional
	Enabled bool `json:"enabled"`
	// The URL that contains the Applications / Application Sets to be used by this capability
	// +kubebuilder:validation:Optional
	GitURL string `json:"gitUrl"`
	// The revision of the git repo that contains the Applications / Application Sets to be used by this capability
	// +kubebuilder:validation:Optional
	GitRevision string `json:"gitRevision"`
	// the path in the git repo that contains the Applications / Application Sets to be used by this capability
	// +kubebuilder:validation:Optional
	GitPath string `json:"gitPath"`
	// Custom fields to configure this specific Capability
	// +kubebuilder:validation:Optional
	CustomFields map[string]string `json:"custom_fields"`
	// This project has its own ClusterResourceQuota settings
	// +kubebuilder:validation:Optional
	Quota paasquota.Quota `json:"quota"`
	// You can add ssh keys (which is a type of secret) for capability to use for access to bitBucket
	// They must be encrypted with a public key, for which the private key should be added to the DecryptKeySecret
	// +kubebuilder:validation:Optional
	SSHSecrets map[string]string `json:"sshSecrets"`
	// You can enable extra permissions for the service accounts belonging to this capability
	// Exact definitions is configured in Paas Configmap
	// +kubebuilder:validation:Optional
	ExtraPermissions bool `json:"extra_permissions"`
}

func (pc *PaasCapability) CapExtraFields(
	fieldConfig map[string]ConfigCustomField,
) (elements fields.Elements, err error) {
	// TODO: remove argocd specific fields
	elements = make(fields.Elements)
	elements["git_url"] = pc.GitURL
	elements["git_revision"] = pc.GitRevision
	elements["git_path"] = pc.GitPath
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

func (pc *PaasCapability) WithExtraPermissions() bool {
	return pc.Enabled && pc.ExtraPermissions
}

// Deprecated: pc.Enabled will be removed
func (pc *PaasCapability) IsEnabled() bool {
	return pc.Enabled
}

func (pc *PaasCapability) SetDefaults() {
	if pc.GitPath == "" {
		pc.GitPath = "."
	}
	if pc.GitRevision == "" {
		pc.GitRevision = "master"
	}
}

func (pc PaasCapability) Quotas() (pq paasquota.Quota) {
	return pc.Quota
}

func (pc PaasCapability) GetSSHSecrets() map[string]string {
	return pc.SSHSecrets
}

func (pc *PaasCapability) SetSSHSecret(key string, value string) {
	pc.SSHSecrets[key] = value
}

// revive:disable:line-length-limit

// PaasStatus defines the observed state of Paas
type PaasStatus struct {
	// Deprecated: use paasns.status.conditions instead
	// +kubebuilder:validation:Optional
	Messages []string `json:"messages"`
	// Deprecated: will not be set and removed in a future release
	// +kubebuilder:validation:Optional
	Quota map[string]paasquota.Quota `json:"quotas"`
	// +kubebuilder:validation:Optional
	Conditions []metav1.Condition `json:"conditions" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// revive:enable:line-length-limit

// Deprecated: use paas.status.conditions instead
func (ps *PaasStatus) Truncate() {
	ps.Messages = []string{}
}

// Deprecated: use paasns.status.conditions instead
func (ps *PaasStatus) GetMessages() []string {
	return ps.Messages
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=paas,scope=Cluster

// Paas is the Schema for the paas API
type Paas struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PaasSpec   `json:"spec,omitempty"`
	Status PaasStatus `json:"status,omitempty"`
}

func (p Paas) ClonedAnnotations() map[string]string {
	annotations := make(map[string]string)
	for key, value := range p.Annotations {
		annotations[key] = value
	}
	return annotations
}

func (p Paas) ClonedLabels() map[string]string {
	labels := make(map[string]string)
	for key, value := range p.Labels {
		if key != "app.kubernetes.io/instance" {
			labels[key] = value
		}
	}
	return labels
}

func (p Paas) IsItMe(reference metav1.OwnerReference) bool {
	if p.APIVersion != reference.APIVersion ||
		p.Kind != reference.Kind ||
		p.Name != reference.Name {
		return false
	}

	return true
}

func (p Paas) AmIOwner(references []metav1.OwnerReference) bool {
	for _, reference := range references {
		if p.IsItMe(reference) {
			return true
		}
	}
	return false
}

func (p Paas) WithoutMe(references []metav1.OwnerReference) (withoutMe []metav1.OwnerReference) {
	for _, reference := range references {
		if !p.IsItMe(reference) {
			withoutMe = append(withoutMe, reference)
		}
	}
	return withoutMe
}

func (p Paas) GetConditions() []metav1.Condition {
	return p.Status.Conditions
}

// +kubebuilder:object:root=true

// PaasList contains a list of Paas
type PaasList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Paas `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Paas{}, &PaasList{})
}
