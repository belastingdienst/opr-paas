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

	"github.com/belastingdienst/opr-paas/internal/groups"
	paas_quota "github.com/belastingdienst/opr-paas/internal/quota"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Definitions to manage status conditions
const (
	// TypeReadyPaas represents the status of the Paas reconciliation
	TypeReadyPaas = "Ready"
	// TypeHasErrorsPaas represents the status used when the Paas reconciliation holds errors.
	TypeHasErrorsPaas = "HasErrors"
	// TypeDegradedPaas represents the status used when the Paas is deleted and the finalizer operations are yet to occur.
	TypeDegradedPaas = "Degraded"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PaasSpec defines the desired state of Paas
type PaasSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Capabilities is a subset of capabilities that will be available in this Paas Project
	Capabilities PaasCapabilities `json:"capabilities,omitempty"`

	// Requestor is an informational field which decides on the requestor (also application responable)
	Requestor string `json:"requestor"`

	Groups PaasGroups `json:"groups,omitempty"`

	// Quota defines the quotas which should be set on the cluster resource quota as used by this Paas project
	Quota paas_quota.Quota `json:"quota"`

	// Namespaces can be used to define extra namespaces to be created as part of this Paas project
	Namespaces []string `json:"namespaces,omitempty"`
	// You can add ssh keys (which is a type of secret) for ArgoCD to use for access to bitBucket
	// They must be encrypted with the public key corresponding to the private key deployed together with the Paas operator
	SshSecrets map[string]string `json:"sshSecrets,omitempty"`

	// Indicated by which 3rd party Paas's ArgoCD this Paas is managed
	ManagedByPaas string `json:"managedByPaas,omitempty"`
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

func (p Paas) GetNsSshSecrets(ns string) (secrets map[string]string) {
	secrets = make(map[string]string)
	for key, value := range p.Spec.SshSecrets {
		secrets[key] = value
	}
	if cap, exists := p.Spec.Capabilities[ns]; exists {
		for key, value := range cap.GetSshSecrets() {
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
	return
}

func (p Paas) AllCapNamespaces() (ns map[string]bool) {
	ns = make(map[string]bool)
	for name := range p.Spec.Capabilities {
		ns[name] = true
	}
	return
}

func (p Paas) PrefixedAllCapNamespaces() (ns map[string]bool) {
	return p.PrefixedBoolMap(p.AllCapNamespaces())
}

func (p Paas) AllEnabledNamespaces() (ns map[string]bool) {
	ns = p.enabledCapNamespaces()
	for name := range p.extraNamespaces() {
		ns[name] = true
	}
	return
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
	return
}

type PaasGroup struct {
	Query string   `json:"query,omitempty"`
	Users []string `json:"users,omitempty"`
	Roles []string `json:"roles,omitempty"`
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

func (pgs PaasGroups) Filtered(groups []string) PaasGroups {
	filtered := make(PaasGroups)
	if len(groups) == 0 {
		return pgs
	}
	for _, groupName := range groups {
		if group, exists := pgs[groupName]; exists {
			filtered[groupName] = group
		}
	}
	return filtered
}

func (pgs PaasGroups) Roles() map[string][]string {
	roles := make(map[string][]string)
	for groupName, group := range pgs {
		roles[groupName] = group.Roles
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

func (pgs PaasGroups) Names() (groups []string) {
	for name, group := range pgs {
		groups = append(groups, group.Name(name))
	}
	return groups
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

func (pgs PaasGroups) Keys() (groups []string) {
	return pgs.Names()
}

func (pgs PaasGroups) AsGroups() groups.Groups {
	newGroups := groups.NewGroups()
	newGroups.AddFromStrings(pgs.LdapQueries())
	return *newGroups
}

// see config/samples/_v1alpha1_paas.yaml for example of CR

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
	if cap, exists := pcs[name]; !exists || !cap.IsEnabled() {
		return false
	}

	return true
}

func (pcs PaasCapabilities) GetCapability(capability string) (cap PaasCapability, err error) {
	if cap, exists := pcs[capability]; !exists {
		return cap, fmt.Errorf("Capability %s does not exist", capability)
	} else {
		return cap, nil
	}
}

func (pcs PaasCapabilities) AddCapSshSecret(capability string, key string, value string) (err error) {
	if cap, err := pcs.GetCapability(capability); err != nil {
		return err
	} else {
		if cap.SshSecrets == nil {
			cap.SshSecrets = map[string]string{key: value}
		} else {
			cap.SshSecrets[key] = value
		}
		pcs[capability] = cap
	}
	return nil
}

func (pcs PaasCapabilities) ResetCapSshSecret(capability string) (err error) {
	if cap, err := pcs.GetCapability(capability); err != nil {
		return err
	} else {
		cap.SshSecrets = nil
		pcs[capability] = cap
	}
	return nil
}

// TODO: Enabled is a leftover from old capability implementation. Remove with new API.

type PaasCapability struct {
	// Do we want to use this capability, default false
	Enabled bool `json:"enabled,omitempty"`
	// The URL that contains the Applications / Application Sets to be used by this capability
	GitUrl string `json:"gitUrl,omitempty"`
	// The revision of the git repo that contains the Applications / Application Sets to be used by this capability
	GitRevision string `json:"gitRevision,omitempty"`
	// the path in the git repo that contains the Applications / Application Sets to be used by this capability
	GitPath string `json:"gitPath,omitempty"`
	// Custom fields to configure this specific Capability
	CustomFields map[string]string `json:"custom_fields,omitempty"`
	// This project has it's own ClusterResourceQuota settings
	Quota paas_quota.Quota `json:"quota,omitempty"`
	// You can add ssh keys (which is a type of secret) for capability to use for access to bitBucket
	// They must be encrypted with the public key corresponding to the private key deployed together with the Paas operator
	SshSecrets map[string]string `json:"sshSecrets,omitempty"`
	// You can enable extra permissions for the service accounts belonging to this capability
	// Exact definitions is configured in Paas Configmap
	// Note that we want to remove (some of) these permissions in future releases (like self-provisioner)
	ExtraPermissions bool `json:"extra_permissions,omitempty"`
}

func (pc *PaasCapability) CapExtraFields(fieldConfig map[string]ConfigCustomField) (fields map[string]string, err error) {
	// TODO: remove argocd specific fields
	fields = map[string]string{
		"git_url":      pc.GitUrl,
		"git_revision": pc.GitRevision,
		"git_path":     pc.GitPath,
	}
	var issues []error
	for key, value := range pc.CustomFields {
		if _, exists := fieldConfig[key]; !exists {
			issues = append(issues, fmt.Errorf("Custom field %s is not configured in capability config", key))
		} else {
			fields[key] = value
		}
	}
	for key, fieldConf := range fieldConfig {
		if value, exists := fields[key]; exists {
			if matched, err := regexp.Match(fieldConf.Validation, []byte(value)); err != nil {
				issues = append(issues, fmt.Errorf("Could not validate value %s: %w", value, err))
			} else if !matched {
				issues = append(issues, fmt.Errorf("Invalid value %s (does not match %s)", value, fieldConf.Validation))
			}
		} else if fieldConf.Required {
			issues = append(issues, fmt.Errorf("Value %s is required", key))
		} else {
			fields[key] = fieldConf.Default
		}
	}
	if len(issues) > 0 {
		return nil, errors.Join(issues...)
	}
	return fields, nil
}

func (pc *PaasCapability) WithExtraPermissions() bool {
	return pc.Enabled && pc.ExtraPermissions
}

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

func (pc PaasCapability) Quotas() (pq paas_quota.Quota) {
	return pc.Quota
}

func (pc PaasCapability) GetSshSecrets() map[string]string {
	return pc.SshSecrets
}

func (pc *PaasCapability) SetSshSecret(key string, value string) {
	pc.SshSecrets[key] = value
}

// PaasStatus defines the observed state of Paas
type PaasStatus struct {
	// Deprecated: use paasns.status.conditions instead
	Messages []string `json:"messages,omitempty"`
	// Deprecated: will not be set and removed in a future release
	Quota      map[string]paas_quota.Quota `json:"quotas,omitempty"`
	Conditions []metav1.Condition          `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// Deprecated: use paas.status.conditions instead
func (ps *PaasStatus) Truncate() {
	ps.Messages = []string{}
}

// Deprecated: use paasns.status.conditions instead
func (ps *PaasStatus) GetMessages() []string {
	return ps.Messages
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=paas,scope=Cluster

// Paas is the Schema for the paas API
type Paas struct {
	metav1.TypeMeta   `json:",inline"`
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

//+kubebuilder:object:root=true

// PaasList contains a list of Paas
type PaasList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Paas `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Paas{}, &PaasList{})
}
