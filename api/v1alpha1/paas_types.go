/*
Copyright 2023.

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

package v1alpha1

import (
	"fmt"
	"strings"

	"github.com/belastingdienst/opr-paas/internal/groups"
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type PaasQuotas map[corev1.ResourceName]resourcev1.Quantity

func (pq PaasQuotas) QuotaWithDefaults(defaults map[string]string) (q PaasQuotas) {
	q = make(PaasQuotas)
	for key, value := range defaults {
		q[corev1.ResourceName(key)] = resourcev1.MustParse(value)
	}
	for key, value := range pq {
		q[key] = value
	}
	return q
}

// PaasSpec defines the desired state of Paas
type PaasSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//Cabailities is a subset of capabilities that will be available in this Pass Project
	Capabilities PaasCapabilities `json:"capabilities,omitempty"`

	//Oplosgroep is an informational field which decides on the oplosgroep that is responsible
	Oplosgroep string `json:"oplosGroep"`

	Groups PaasGroups `json:"groups,omitempty"`

	// Quota defines the quotas which should be set on the cluster resource quota as used by this PaaS project
	Quota PaasQuotas `json:"quota"`

	// Namespaces can be used to define extra namespaces to be created as part of this PaaS project
	Namespaces []string `json:"namespaces,omitempty"`
	// You can add ssh keys (which is a type of secret) for ArgoCD to use for access to bitBucket
	// They must be encrypted with the public key corresponding to the private key deployed together with the PaaS operator
	SshSecrets map[string]string `json:"sshSecrets,omitempty"`
}

func (p Paas) EnabledCapNamespaces() (ns map[string]bool) {
	ns = make(map[string]bool)
	for name, cap := range p.Spec.Capabilities.AsMap() {
		if cap.IsEnabled() {
			name = fmt.Sprintf("%s-%s", p.Name, name)
			ns[name] = true
		}
	}
	return
}
func (p Paas) AllCapNamespaces() (ns map[string]bool) {
	ns = make(map[string]bool)
	for name := range p.Spec.Capabilities.AsMap() {
		name = fmt.Sprintf("%s-%s", p.Name, name)
		ns[name] = true
	}
	return
}

func (p Paas) AllEnabledNamespaces() (ns map[string]bool) {
	ns = p.EnabledCapNamespaces()
	for name := range p.ExtraNamespaces() {
		ns[name] = true
	}
	return

}

func (p Paas) ExtraNamespaces() (ns map[string]bool) {
	capNs := p.AllCapNamespaces()
	ns = make(map[string]bool)
	for _, name := range p.Spec.Namespaces {
		name = fmt.Sprintf("%s-%s", p.Name, name)
		if _, isCap := capNs[name]; !isCap {
			ns[name] = true
		}
	}
	return

}

func (p Paas) InvalidExtraNamespaces() (ns map[string]bool) {
	ns = make(map[string]bool)
	capNs := p.AllCapNamespaces()
	for _, name := range p.Spec.Namespaces {
		name = fmt.Sprintf("%s-%s", p.Name, name)
		if _, isCap := capNs[name]; isCap {
			ns[name] = true
		}
	}
	return
}

type PaasGroup struct {
	Query string   `json:"query,omitempty"`
	Users []string `json:"users,omitempty"`
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

// NameFromQuery finds a group by its key, and retrieves a name
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

func (pgs PaasGroups) Keys() (groups []string) {
	for key, group := range pgs {
		groups = append(groups, group.Name(key))
	}
	return groups
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

type PaasCapability interface {
	IsEnabled() bool
	Quotas() PaasQuotas
	CapabilityName() string
	GetSshSecrets() map[string]string
	WithExtraPermissions() bool
}

func (pc PaasCapabilities) AsMap() map[string]PaasCapability {
	caps := make(map[string]PaasCapability)
	for _, cap := range []PaasCapability{
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
	// This project has it's own ClusterResourceQuota seetings
	Quota PaasQuotas `json:"quota,omitempty"`
	// You can add ssh keys (which is a type of secret) for ArgoCD to use for access to bitBucket
	// They must be encrypted with the public key corresponding to the private key deployed together with the PaaS operator
	SshSecrets map[string]string `json:"sshSecrets,omitempty"`
	// You can enable extra permissions for the service accounts beloning to this capability
	// Exact definitions is configured in Paas Configmap
	// Note that we want to remove (some of) these permissions in future releases (like self-provisioner)
	ExtraPermissions bool `json:"extra_permissions,omitempty"`
}

func (pa *PaasArgoCD) WithExtraPermissions() bool {
	return (pa.Enabled && pa.ExtraPermissions)
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

func (pa PaasArgoCD) Quotas() (pq PaasQuotas) {
	return pa.Quota
}

func (pa PaasArgoCD) GetSshSecrets() map[string]string {
	return pa.SshSecrets
}

type PaasCI struct {
	// Do we want a CI (Tekton) namespace, default false
	Enabled bool `json:"enabled,omitempty"`
	// This project has it's own ClusterResourceQuota seetings
	Quota PaasQuotas `json:"quota,omitempty"`
	// You can add ssh keys (which is a type of secret) for ArgoCD to use for access to bitBucket
	// They must be encrypted with the public key corresponding to the private key deployed together with the PaaS operator
	SshSecrets map[string]string `json:"sshSecrets,omitempty"`
	// You can enable extra permissions for the service accounts beloning to this capability
	// Exact definitions is configured in Paas Configmap
	// Note that we want to remove (some of) these permissions in future releases (like self-provisioner)
	ExtraPermissions bool `json:"extra_permissions,omitempty"`
}

func (pc *PaasCI) WithExtraPermissions() bool {
	return (pc.Enabled && pc.ExtraPermissions)
}

func (pc PaasCI) Quotas() (pq PaasQuotas) {
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

type PaasSSO struct {
	// Do we want an SSO namespace, default false
	Enabled bool `json:"enabled,omitempty"`
	// This project has it's own ClusterResourceQuota seetings
	Quota PaasQuotas `json:"quota,omitempty"`
	// You can add ssh keys (which is a type of secret) for ArgoCD to use for access to bitBucket
	// They must be encrypted with the public key corresponding to the private key deployed together with the PaaS operator
	SshSecrets map[string]string `json:"sshSecrets,omitempty"`
	// You can enable extra permissions for the service accounts beloning to this capability
	// Exact definitions is configured in Paas Configmap
	// Note that we want to remove (some of) these permissions in future releases (like self-provisioner)
	ExtraPermissions bool `json:"extra_permissions,omitempty"`
}

func (ps *PaasSSO) WithExtraPermissions() bool {
	return (ps.Enabled && ps.ExtraPermissions)
}

func (ps PaasSSO) Quotas() (pq PaasQuotas) {
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

type PaasGrafana struct {
	// Do we want a Grafana namespace, default false
	Enabled bool `json:"enabled,omitempty"`
	// This project has it's own ClusterResourceQuota seetings
	Quota PaasQuotas `json:"quota,omitempty"`
	// You can add ssh keys (which is a type of secret) for ArgoCD to use for access to bitBucket
	// They must be encrypted with the public key corresponding to the private key deployed together with the PaaS operator
	SshSecrets map[string]string `json:"sshSecrets,omitempty"`
	// You can enable extra permissions for the service accounts beloning to this capability
	// Exact definitions is configured in Paas Configmap
	// Note that we want to remove (some of) these permissions in future releases (like self-provisioner)
	ExtraPermissions bool `json:"extra_permissions,omitempty"`
}

func (pg *PaasGrafana) WithExtraPermissions() bool {
	return (pg.Enabled && pg.ExtraPermissions)
}

func (pg PaasGrafana) Quotas() (pq PaasQuotas) {
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

// PaasStatus defines the observed state of Paas
type PaasStatus struct {
	// Important: Run "make" to regenerate code after modifying this file
	ArgoCDUrl  string   `json:"argocdUrl,omitempty"`
	GrafanaUrl string   `json:"grafanaUrl,omitempty"`
	Messages   []string `json:"messages,omitempty"`
}

func (ps *PaasStatus) Truncate() {
	ps.Messages = []string{}
}

type PaasStatusLevel string
type PaasStatusAction string

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

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:resource:path=paas,scope=Cluster

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
		labels[key] = value
	}
	return labels
}

func (p Paas) IsItMe(reference metav1.OwnerReference) bool {
	if p.APIVersion != reference.APIVersion {
		return false
	} else if p.Kind != reference.Kind {
		return false
	} else if p.Name != reference.Name {
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
