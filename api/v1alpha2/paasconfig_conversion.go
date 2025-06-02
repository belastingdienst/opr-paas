/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha2

import (
	"fmt"
	"maps"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/rs/zerolog/log"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ---------- ConvertTo

// ConvertTo converts this PaasConfig (v1alpha2) to the Hub version (v1alpha1).
func (pc *PaasConfig) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*v1alpha1.PaasConfig)
	if !ok {
		return fmt.Errorf("cannot convert to v1alpha1: got %T", dstRaw)
	}

	logger := log.With().
		Any("conversion", pc.GetObjectKind().GroupVersionKind()).
		Str("name", pc.GetName()).
		Logger()
	logger.Debug().Msg("Starting conversion from spoke (v1alpha2) to hub (v1alpha1)")

	dst.ObjectMeta = pc.ObjectMeta
	dst.Status.Conditions = pc.Status.Conditions
	dst.Spec = v1alpha1.PaasConfigSpec{}
	spec := &dst.Spec

	spec.DecryptKeysSecret = pc.Spec.DecryptKeysSecret.ConvertTo()
	spec.Debug = pc.Spec.Debug

	// Convert configcapability
	spec.Capabilities = make(map[string]v1alpha1.ConfigCapability, len(pc.Spec.Capabilities))
	for key, val := range pc.Spec.Capabilities {
		spec.Capabilities[key] = val.ConvertTo()
	}

	spec.GroupSyncList = v1alpha1.NamespacedName{}
	spec.GroupSyncListKey = ""
	spec.LDAP = v1alpha1.ConfigLdap{}
	spec.ArgoPermissions = v1alpha1.ConfigArgoPermissions{}
	spec.ArgoEnabled = false
	spec.ClusterWideArgoCDNamespace = pc.Spec.ClusterWideArgoCDNamespace
	spec.QuotaLabel = pc.Spec.QuotaLabel
	spec.RequestorLabel = pc.Spec.RequestorLabel
	spec.ManagedByLabel = pc.Spec.ManagedByLabel
	spec.ManagedBySuffix = pc.Spec.ManagedBySuffix
	spec.ExcludeAppSetName = ""
	spec.RoleMappings = pc.Spec.RoleMappings.convertTo()
	spec.Validations = pc.Spec.Validations.convertTo()

	return nil
}

// ConvertTo converts a v1alpha2 NamespacedName into a v1alpha1 NamespacedName.
func (nn *NamespacedName) ConvertTo() v1alpha1.NamespacedName {
	return v1alpha1.NamespacedName{
		Namespace: nn.Namespace,
		Name:      nn.Name,
	}
}

// ConvertTo converts a v1alpha2 ConfigCapability into a v1alpha1 ConfigCapability.
func (cc *ConfigCapability) ConvertTo() v1alpha1.ConfigCapability {
	dst := v1alpha1.ConfigCapability{}
	dst.AppSet = cc.AppSet
	dst.DefaultPermissions = v1alpha1.ConfigCapPerm(cc.DefaultPermissions)
	dst.ExtraPermissions = v1alpha1.ConfigCapPerm(cc.ExtraPermissions)
	dst.QuotaSettings = cc.QuotaSettings.convertTo()

	// Convert customfields
	dst.CustomFields = make(map[string]v1alpha1.ConfigCustomField, len(cc.CustomFields))
	for key, val := range cc.CustomFields {
		// Use the method defined on ConfigCustomField
		dst.CustomFields[key] = val.ConvertTo()
	}

	return dst
}

// convertTo converts a v1alpha2.ConfigQuotaSettings to v1alpha1.ConfigQuotaSettings.
func (cqs *ConfigQuotaSettings) convertTo() v1alpha1.ConfigQuotaSettings {
	return v1alpha1.ConfigQuotaSettings{
		Clusterwide: cqs.Clusterwide,
		Ratio:       cqs.Ratio,
		DefQuota:    cqs.DefQuota,
		MinQuotas:   cqs.MinQuotas,
		MaxQuotas:   cqs.MaxQuotas,
	}
}

// ConvertTo converts a v1alpha2.ConfigCustomField to v1alpha1.ConfigCustomField.
func (ccf *ConfigCustomField) ConvertTo() v1alpha1.ConfigCustomField {
	return v1alpha1.ConfigCustomField{
		Validation: ccf.Validation,
		Default:    ccf.Default,
		Template:   ccf.Template,
		Required:   ccf.Required,
	}
}

// convertTo converts a v1alpha2.ConfigRoleMappings to v1alpha1.ConfigRoleMappings.
func (crm ConfigRoleMappings) convertTo() v1alpha1.ConfigRoleMappings {
	if crm == nil {
		return nil
	}

	out := make(v1alpha1.ConfigRoleMappings, len(crm))
	for key, list := range crm {
		if list == nil {
			out[key] = nil
			continue
		}
		// Deep copy the slice
		copied := make([]string, len(list))
		copy(copied, list)
		out[key] = copied
	}
	return out
}

// convertTo converts a v1alpha2.PaasConfigValidations to v1alpha1.PaasConfigValidations.
func (pcv PaasConfigValidations) convertTo() v1alpha1.PaasConfigValidations {
	if pcv == nil {
		return nil
	}

	out := make(v1alpha1.PaasConfigValidations, len(pcv))
	for key, inner := range pcv {
		if inner == nil {
			out[key] = nil
			continue
		}
		copied := make(v1alpha1.PaasConfigTypeValidations, len(inner))
		maps.Copy(copied, inner)
		out[key] = copied
	}
	return out
}

// ------ ConvertFrom

// ConvertFrom converts from the Hub version (v1alpha1) to this version (v1alpha2).
func (pc *PaasConfig) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*v1alpha1.PaasConfig)
	if !ok {
		return fmt.Errorf("cannot convert from %T: expected v1alpha1.PaasConfig", srcRaw)
	}

	logger := log.With().
		Any("conversion", src.GetObjectKind().GroupVersionKind()).
		Str("name", src.GetName()).
		Logger()
	logger.Debug().Msg("Starting conversion from hub (v1alpha1) to spoke (v1alpha2)")

	pc.ObjectMeta = src.ObjectMeta
	pc.Status.Conditions = src.Status.Conditions
	pc.Spec = PaasConfigSpec{}
	spec := &pc.Spec

	spec.DecryptKeysSecret = spec.DecryptKeysSecret.ConvertFrom(src.Spec.DecryptKeysSecret)
	spec.Debug = src.Spec.Debug

	spec.Capabilities = make(map[string]ConfigCapability, len(src.Spec.Capabilities))
	for key, val := range src.Spec.Capabilities {
		spec.Capabilities[key] = ConfigCapability{}.convertFrom(val)
	}

	spec.ClusterWideArgoCDNamespace = src.Spec.ClusterWideArgoCDNamespace
	spec.QuotaLabel = src.Spec.QuotaLabel
	spec.RequestorLabel = src.Spec.RequestorLabel
	spec.ManagedByLabel = src.Spec.ManagedByLabel
	spec.ManagedBySuffix = src.Spec.ManagedBySuffix
	spec.RoleMappings = ConfigRoleMappings{}.convertFrom(src.Spec.RoleMappings)
	spec.Validations = PaasConfigValidations{}.convertFrom(src.Spec.Validations)

	return nil
}

// ConvertFrom converts the given v1alpha1.NamespacedName to v1alpha2.NamespacedName
func (nn *NamespacedName) ConvertFrom(n v1alpha1.NamespacedName) NamespacedName {
	return NamespacedName{
		Namespace: n.Namespace,
		Name:      n.Name,
	}
}

func (cc ConfigCapability) convertFrom(src v1alpha1.ConfigCapability) ConfigCapability {
	dst := ConfigCapability{}
	dst.AppSet = src.AppSet
	dst.DefaultPermissions = ConfigCapPerm(src.DefaultPermissions)
	dst.ExtraPermissions = ConfigCapPerm(src.ExtraPermissions)
	dst.QuotaSettings = dst.QuotaSettings.convertFrom(src.QuotaSettings)

	dst.CustomFields = make(map[string]ConfigCustomField, len(src.CustomFields))
	for key, val := range src.CustomFields {
		dst.CustomFields[key] = ConfigCustomField{}.convertFrom(val)
	}
	return dst
}

func (cqs ConfigQuotaSettings) convertFrom(src v1alpha1.ConfigQuotaSettings) ConfigQuotaSettings {
	return ConfigQuotaSettings{
		Clusterwide: src.Clusterwide,
		Ratio:       src.Ratio,
		DefQuota:    src.DefQuota,
		MinQuotas:   src.MinQuotas,
		MaxQuotas:   src.MaxQuotas,
	}
}

func (ccf ConfigCustomField) convertFrom(src v1alpha1.ConfigCustomField) ConfigCustomField {
	return ConfigCustomField{
		Validation: src.Validation,
		Default:    src.Default,
		Template:   src.Template,
		Required:   src.Required,
	}
}

func (crm ConfigRoleMappings) convertFrom(src v1alpha1.ConfigRoleMappings) ConfigRoleMappings {
	if src == nil {
		return nil
	}

	dst := make(ConfigRoleMappings, len(src))
	for k, v := range src {
		dst[k] = append([]string(nil), v...)
	}
	return dst
}

func (pcv PaasConfigValidations) convertFrom(src v1alpha1.PaasConfigValidations) PaasConfigValidations {
	if src == nil {
		return nil
	}

	dst := make(PaasConfigValidations, len(src))
	for key, inner := range src {
		if inner == nil {
			dst[key] = nil
			continue
		}
		copied := make(PaasConfigTypeValidations, len(inner))
		maps.Copy(copied, inner)
		dst[key] = copied
	}
	return dst
}
