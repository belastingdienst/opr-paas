/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"fmt"
	"strings"

	"github.com/belastingdienst/opr-paas/internal/groups"
	paas_quota "github.com/belastingdienst/opr-paas/internal/quota"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PaasSpec defines the desired state of Paas
type PaasSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Capabilities is a subset of capabilities that will be available in this PaaS Project
	Capabilities PaasCapabilities `json:"capabilities,omitempty"`

	// Requestor is an informational field which decides on the requestor (also application responable)
	Requestor string `json:"requestor"`

	Groups PaasGroups `json:"groups,omitempty"`

	// Quota defines the quotas which should be set on the cluster resource quota as used by this PaaS project
	Quota paas_quota.Quotas `json:"quota"`

	// Namespaces can be used to define extra namespaces to be created as part of this PaaS project
	Namespaces []string `json:"namespaces,omitempty"`
	// You can add ssh keys (which is a type of secret) for ArgoCD to use for access to bitBucket
	// They must be encrypted with the public key corresponding to the private key deployed together with the PaaS operator
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
	if cap, exists := p.Spec.Capabilities.AsMap()[ns]; exists {
		for key, value := range cap.GetSshSecrets() {
			secrets[key] = value
		}
	}
	return secrets
}

func (p Paas) enabledCapNamespaces() (ns map[string]bool) {
	ns = make(map[string]bool)
	for name, cap := range p.Spec.Capabilities.AsMap() {
		if cap.IsEnabled() {
			ns[name] = true
		}
	}
	return
}

func (p Paas) AllCapNamespaces() (ns map[string]bool) {
	ns = make(map[string]bool)
	for name := range p.Spec.Capabilities.AsMap() {
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

// func (p Paas) invalidExtraNamespaces() (ns map[string]bool) {
// 	ns = make(map[string]bool)
// 	capNs := p.AllCapNamespaces()
// 	for _, name := range p.Spec.Namespaces {
// 		name = fmt.Sprintf("%s-%s", p.Name, name)
// 		if _, isCap := capNs[name]; isCap {
// 			ns[name] = true
// 		}
// 	}
// 	return
// }

type PaasGroup struct {
	Query string   `json:"query,omitempty"`
	Users []string `json:"users,omitempty"`
	Roles []string `json:"roles,omitempty"`
}

func (g PaasGroup) Name(defName string) string {
	if name := strings.Split(g.Query, ",")[0]; len(name) == 0 {
		return defName
	} else if strings.Contains(name, "=") {
		return strings.Split(name, "=")[1]
	} else {
		return name
	}
}

type PaasGroups map[string]PaasGroup

func (gs PaasGroups) Filtered(groups []string) PaasGroups {
	filtered := make(PaasGroups)
	if len(groups) == 0 {
		return gs
	}
	for _, groupName := range groups {
		if group, exists := gs[groupName]; exists {
			filtered[groupName] = group
		}
	}
	return filtered
}

func (gs PaasGroups) Roles() map[string][]string {
	roles := make(map[string][]string)
	for groupName, group := range gs {
		roles[groupName] = group.Roles
	}
	return roles
}

// Key2Name finds a group by its key, and retrieves a name
// - from query if possible
// - from key is needed
// - emptystring if not in map
func (g PaasGroups) Key2Name(key string) string {
	if group, exists := g[key]; !exists {
		return ""
	} else {
		return group.Name(key)
	}
}

func (gs PaasGroups) Names() (groups []string) {
	for name, group := range gs {
		groups = append(groups, group.Name(name))
	}
	return groups
}

func (gs PaasGroups) LdapQueries() []string {
	var queries []string
	for _, group := range gs {
		if group.Query != "" {
			queries = append(queries, group.Query)
		}
	}
	return queries
}

// TODO: remove either Keys() or Names()? - were duplicate functions
func (pgs PaasGroups) Keys() (groups []string) {
	return pgs.Names()
}

func (pgs PaasGroups) AsGroups() groups.Groups {
	gs := groups.NewGroups()
	gs.AddFromStrings(pgs.LdapQueries())
	return *gs
}

// see config/samples/_v1alpha1_paas.yaml for example of CR

type PaasCapabilities struct {
	// ArgoCD defines the ArgoCD deployment that should be available.
	ArgoCD PaasArgoCD `json:"argocd,omitempty"`
	// CI defines the settings for a CI namespace (tekton) for this PAAS
	CI PaasCI `json:"tekton,omitempty"`
	// SSO defines the settings for a SSO (KeyCloak) namwespace for this PAAS
	SSO PaasSSO `json:"sso,omitempty"`
	// Grafana defines the settings for a Grafana monitoring namespace for this PAAS
	Grafana PaasGrafana `json:"grafana,omitempty"`
}

type paasCapability interface {
	IsEnabled() bool
	Quotas() paas_quota.Quotas
	CapabilityName() string
	GetSshSecrets() map[string]string
	SetSshSecret(string, string)
	WithExtraPermissions() bool
}

/*
AsMap geeft de namen van de capabilties, terwijl bijvoorbeeld de namespace namen en quota namen geprefixt zijn met de paas naam.
Daarom een AsPrefixedMap, zodat we ook makkelijk kunnen zoeken als je de namespace naam hebt.
*/
func (pc PaasCapabilities) AsPrefixedMap(prefix string) map[string]paasCapability {
	if prefix == "" {
		return pc.AsMap()
	}
	caps := make(map[string]paasCapability)
	for name, cap := range pc.AsMap() {
		caps[fmt.Sprintf("%s-%s", prefix, name)] = cap
	}
	return caps
}

func (pc PaasCapabilities) IsCap(name string) bool {
	caps := pc.AsMap()

	if cap, exists := caps[name]; !exists || !cap.IsEnabled() {
		return false
	}

	return true
}

func (pc PaasCapabilities) AsMap() map[string]paasCapability {
	caps := make(map[string]paasCapability)
	for _, cap := range []paasCapability{
		&pc.ArgoCD,
		&pc.CI,
		&pc.SSO,
		&pc.Grafana,
	} {
		caps[cap.CapabilityName()] = cap
	}
	return caps
}

type PaasArgoCD struct {
	// Do we want an ArgoCD namespace, default false
	Enabled bool `json:"enabled,omitempty"`
	// The URL that contains the Applications / Application Sets to be used by this ArgoCD
	GitUrl string `json:"gitUrl,omitempty"`
	// The revision of the git repo that contains the Applications / Application Sets to be used by this ArgoCD
	GitRevision string `json:"gitRevision,omitempty"`
	// the path in the git repo that contains the Applications / Application Sets to be used by this ArgoCD
	GitPath string `json:"gitPath,omitempty"`
	// This project has it's own ClusterResourceQuota settings
	Quota paas_quota.Quotas `json:"quota,omitempty"`
	// You can add ssh keys (which is a type of secret) for ArgoCD to use for access to bitBucket
	// They must be encrypted with the public key corresponding to the private key deployed together with the PaaS operator
	SshSecrets map[string]string `json:"sshSecrets,omitempty"`
	// You can enable extra permissions for the service accounts beloning to this capability
	// Exact definitions is configured in Paas Configmap
	// Note that we want to remove (some of) these permissions in future releases (like self-provisioner)
	ExtraPermissions bool `json:"extra_permissions,omitempty"`
}

func (pa *PaasArgoCD) WithExtraPermissions() bool {
	return pa.Enabled && pa.ExtraPermissions
}

func (pa *PaasArgoCD) IsEnabled() bool {
	return pa.Enabled
}

func (pa *PaasArgoCD) CapabilityName() string {
	return "argocd"
}

func (pa *PaasArgoCD) SetDefaults() {
	if pa.GitPath == "" {
		pa.GitPath = "."
	}
	if pa.GitRevision == "" {
		pa.GitRevision = "master"
	}
}

func (pa PaasArgoCD) Quotas() (pq paas_quota.Quotas) {
	return pa.Quota
}

func (pa PaasArgoCD) GetSshSecrets() map[string]string {
	return pa.SshSecrets
}

func (pa *PaasArgoCD) SetSshSecret(key string, value string) {
	pa.SshSecrets[key] = value
}

type PaasCI struct {
	// Do we want a CI (Tekton) namespace, default false
	Enabled bool `json:"enabled,omitempty"`
	// This project has it's own ClusterResourceQuota settings
	Quota paas_quota.Quotas `json:"quota,omitempty"`
	// You can add ssh keys (which is a type of secret) for ArgoCD to use for access to bitBucket
	// They must be encrypted with the public key corresponding to the private key deployed together with the PaaS operator
	SshSecrets map[string]string `json:"sshSecrets,omitempty"`
	// You can enable extra permissions for the service accounts beloning to this capability
	// Exact definitions is configured in Paas Configmap
	// Note that we want to remove (some of) these permissions in future releases (like self-provisioner)
	ExtraPermissions bool `json:"extra_permissions,omitempty"`
}

func (pc *PaasCI) WithExtraPermissions() bool {
	return pc.Enabled && pc.ExtraPermissions
}

func (pc PaasCI) Quotas() (pq paas_quota.Quotas) {
	return pc.Quota
}

func (pc *PaasCI) IsEnabled() bool {
	return pc.Enabled
}

func (pc *PaasCI) CapabilityName() string {
	return "tekton"
}

func (pc PaasCI) GetSshSecrets() map[string]string {
	return pc.SshSecrets
}

func (pc *PaasCI) SetSshSecret(key string, value string) {
	pc.SshSecrets[key] = value
}

type PaasSSO struct {
	// Do we want an SSO namespace, default false
	Enabled bool `json:"enabled,omitempty"`
	// This project has its own ClusterResourceQuota settings
	Quota paas_quota.Quotas `json:"quota,omitempty"`
	// You can add ssh keys (which is a type of secret) for ArgoCD to use for access to bitBucket
	// They must be encrypted with the public key corresponding to the private key deployed together with the PaaS operator
	SshSecrets map[string]string `json:"sshSecrets,omitempty"`
	// You can enable extra permissions for the service accounts beloning to this capability
	// Exact definitions is configured in Paas Configmap
	// Note that we want to remove (some of) these permissions in future releases (like self-provisioner)
	ExtraPermissions bool `json:"extra_permissions,omitempty"`
}

func (ps *PaasSSO) WithExtraPermissions() bool {
	return ps.Enabled && ps.ExtraPermissions
}

func (ps PaasSSO) Quotas() (pq paas_quota.Quotas) {
	return ps.Quota
}

func (ps *PaasSSO) IsEnabled() bool {
	return ps.Enabled
}

func (ps *PaasSSO) CapabilityName() string {
	return "sso"
}

func (ps PaasSSO) GetSshSecrets() map[string]string {
	return ps.SshSecrets
}

func (ps *PaasSSO) SetSshSecret(key string, value string) {
	ps.SshSecrets[key] = value
}

type PaasGrafana struct {
	// Do we want a Grafana namespace, default false
	Enabled bool `json:"enabled,omitempty"`
	// This project has it's own ClusterResourceQuota settings
	Quota paas_quota.Quotas `json:"quota,omitempty"`
	// You can add ssh keys (which is a type of secret) for ArgoCD to use for access to bitBucket
	// They must be encrypted with the public key corresponding to the private key deployed together with the PaaS operator
	SshSecrets map[string]string `json:"sshSecrets,omitempty"`
	// You can enable extra permissions for the service accounts beloning to this capability
	// Exact definitions is configured in Paas Configmap
	// Note that we want to remove (some of) these permissions in future releases (like self-provisioner)
	ExtraPermissions bool `json:"extra_permissions,omitempty"`
}

func (pg *PaasGrafana) WithExtraPermissions() bool {
	return pg.Enabled && pg.ExtraPermissions
}

func (pg PaasGrafana) Quotas() (pq paas_quota.Quotas) {
	return pg.Quota
}

func (pg *PaasGrafana) IsEnabled() bool {
	return pg.Enabled
}

func (pg *PaasGrafana) CapabilityName() string {
	return "grafana"
}

func (pg PaasGrafana) GetSshSecrets() map[string]string {
	return pg.SshSecrets
}

func (pg *PaasGrafana) SetSshSecret(key string, value string) {
	pg.SshSecrets[key] = value
}

// PaasStatus defines the observed state of Paas
type PaasStatus struct {
	// Important: Run "make" to regenerate code after modifying this file
	Messages []string                     `json:"messages,omitempty"`
	Quota    map[string]paas_quota.Quotas `json:"quotas"`
}

func (ps *PaasStatus) Truncate() {
	ps.Messages = []string{}
}

type (
	PaasStatusLevel  string
	PaasStatusAction string
)

const (
	PaasStatusInfo      PaasStatusLevel  = "INFO"
	PaasStatusWarning   PaasStatusLevel  = "WARNING"
	PaasStatusError     PaasStatusLevel  = "ERROR"
	PaasStatusParse     PaasStatusAction = "parse"
	PaasStatusCreate    PaasStatusAction = "create"
	PaasStatusDelete    PaasStatusAction = "delete"
	PaasStatusFind      PaasStatusAction = "find"
	PaasStatusUpdate    PaasStatusAction = "update"
	PaasStatusReconcile PaasStatusAction = "reconcile"
)

func (ps *PaasStatus) AddMessage(level PaasStatusLevel, action PaasStatusAction, obj client.Object, message string) {
	namespacedName := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
	ps.Messages = append(ps.Messages,
		fmt.Sprintf("%s: %s for %s (%s) %s", level, action, namespacedName.String(), obj.GetObjectKind().GroupVersionKind().String(), message),
	)
}

func (ps *PaasStatus) GetMessages() []string {
	return ps.Messages
}

func (ps *PaasStatus) AddMessages(msgs []string) {
	ps.Messages = append(ps.Messages, msgs...)
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
