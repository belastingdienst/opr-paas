/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha2

import (
	"fmt"
	"sort"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

const (
	gitUrlKey      = "git_url"
	gitRevisionKey = "git_revision"
	gitPathKey     = "git_path"
)

// ConvertTo converts this Paas (v1alpha2) to the Hub version (v1alpha1).
func (p *Paas) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*v1alpha1.Paas)
	if !ok {
		return fmt.Errorf("cannot convert to %s/%s: must be v1alpha1", dst.Namespace, dst.Name)
	}

	dst.ObjectMeta = p.ObjectMeta
	dst.Status.Conditions = p.Status.Conditions
	dst.Spec.Requestor = p.Spec.Requestor
	dst.Spec.Quota = p.Spec.Quota
	dst.Spec.Capabilities = make(v1alpha1.PaasCapabilities)
	dst.Spec.Groups = make(v1alpha1.PaasGroups)
	dst.Spec.Namespaces = make([]string, 0)
	dst.Spec.SSHSecrets = p.Spec.Secrets
	dst.Spec.ManagedByPaas = p.Spec.ManagedByPaas

	for name, capability := range p.Spec.Capabilities {
		fields := capability.DeepCopy().CustomFields
		gitUrl := fields[gitUrlKey]
		gitRevision := fields[gitRevisionKey]
		gitPath := fields[gitPathKey]
		delete(fields, gitUrlKey)
		delete(fields, gitRevisionKey)
		delete(fields, gitPathKey)

		dst.Spec.Capabilities[name] = v1alpha1.PaasCapability{
			Enabled:          true,
			GitURL:           gitUrl,
			GitRevision:      gitRevision,
			GitPath:          gitPath,
			CustomFields:     fields,
			Quota:            capability.Quota,
			SSHSecrets:       capability.Secrets,
			ExtraPermissions: capability.ExtraPermissions,
		}
	}

	for name, group := range p.Spec.Groups {
		dst.Spec.Groups[name] = v1alpha1.PaasGroup{
			Query: group.Query,
			Users: group.Users,
			Roles: group.Roles,
		}
	}

	for name := range p.Spec.Namespaces {
		dst.Spec.Namespaces = append(dst.Spec.Namespaces, name)
	}
	sort.Strings(dst.Spec.Namespaces)

	return nil
}

// ConvertFrom converts the Hub version (v1alpha1) to this Paas (v1alpha2).
func (p *Paas) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*v1alpha1.Paas)
	if !ok {
		return fmt.Errorf("cannot convert %s/%s: must be v1alpha1", src.Namespace, src.Name)
	}

	p.ObjectMeta = src.ObjectMeta
	p.Status.Conditions = src.Status.Conditions
	p.Spec.Requestor = src.Spec.Requestor
	p.Spec.Quota = src.Spec.Quota
	p.Spec.Capabilities = make(PaasCapabilities)
	p.Spec.Groups = make(PaasGroups)
	p.Spec.Namespaces = make(PaasNamespaces)
	p.Spec.Secrets = src.Spec.SSHSecrets
	p.Spec.ManagedByPaas = src.Spec.ManagedByPaas

	for name, capability := range src.Spec.Capabilities {
		fields := make(map[string]string)
		for f := range capability.CustomFields {
			fields[f] = capability.CustomFields[f]
		}
		if capability.GitURL != "" {
			fields[gitUrlKey] = capability.GitURL
		}
		if capability.GitRevision != "" {
			fields[gitRevisionKey] = capability.GitRevision
		}
		if capability.GitPath != "" {
			fields[gitPathKey] = capability.GitPath
		}

		p.Spec.Capabilities[name] = PaasCapability{
			CustomFields:     fields,
			Quota:            capability.Quota,
			Secrets:          capability.SSHSecrets,
			ExtraPermissions: capability.ExtraPermissions,
		}
	}

	for name, group := range src.Spec.Groups {
		p.Spec.Groups[name] = PaasGroup{
			Query: group.Query,
			Users: group.Users,
			Roles: group.Roles,
		}
	}

	for _, name := range src.Spec.Namespaces {
		p.Spec.Namespaces[name] = PaasNamespace{}
	}

	return nil
}
