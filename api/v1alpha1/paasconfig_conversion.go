/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"fmt"
	"maps"

	"github.com/belastingdienst/opr-paas/api/v1alpha2"
	"github.com/rs/zerolog/log"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ------ ConvertFrom

// ConvertFrom converts from the Hub version (v1alpha2) to this version (v1alpha1).
func (pc *PaasConfig) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*v1alpha2.PaasConfig)
	if !ok {
		return fmt.Errorf("cannot convert to v1alpha1: got %T", srcRaw)
	}

	logger := log.With().
		Any("conversion", src.GetObjectKind().GroupVersionKind()).
		Str("name", src.GetName()).
		Logger()
	logger.Debug().Msg("Starting conversion from hub (v1alpha2) to spoke (v1alpha1)")

	pc.ObjectMeta = src.ObjectMeta
	pc.Status.Conditions = src.Status.Conditions
	pc.Spec = PaasConfigSpec{}
	spec := &pc.Spec

	spec.DecryptKeysSecret = convertFromNamespacedName(src.Spec.DecryptKeysSecret)
	spec.Debug = src.Spec.Debug

	// Convert configcapability
	spec.Capabilities = make(map[string]ConfigCapability, len(src.Spec.Capabilities))
	for key, val := range src.Spec.Capabilities {
		spec.Capabilities[key] = convertFromConfigCapability(val)
	}

	spec.GroupSyncList = NamespacedName{}
	spec.GroupSyncListKey = ""
	spec.LDAP = ConfigLdap{}
	spec.ArgoPermissions = ConfigArgoPermissions{}
	spec.ArgoEnabled = false
	spec.ClusterWideArgoCDNamespace = src.Spec.ClusterWideArgoCDNamespace
	spec.QuotaLabel = src.Spec.QuotaLabel
	spec.RequestorLabel = src.Spec.RequestorLabel
	spec.ManagedByLabel = src.Spec.ManagedByLabel
	spec.ManagedBySuffix = src.Spec.ManagedBySuffix
	spec.ExcludeAppSetName = ""
	spec.RoleMappings = convertFromRoleMappings(src.Spec.RoleMappings)
	spec.Validations = convertFromValidations(src.Spec.Validations)

	return nil
}

// convertFromNamespacedName converts a v1alpha2 NamespacedName into a v1alpha1 NamespacedName.
func convertFromNamespacedName(nn v1alpha2.NamespacedName) NamespacedName {
	return NamespacedName{
		Namespace: nn.Namespace,
		Name:      nn.Name,
	}
}

// convertFromConfigCapability converts a v1alpha2 ConfigCapability into a v1alpha1 ConfigCapability.
func convertFromConfigCapability(cc v1alpha2.ConfigCapability) ConfigCapability {
	dst := ConfigCapability{}
	dst.AppSet = cc.AppSet
	dst.DefaultPermissions = ConfigCapPerm(cc.DefaultPermissions)
	dst.ExtraPermissions = ConfigCapPerm(cc.ExtraPermissions)
	dst.QuotaSettings = convertFromQuotaSettings(cc.QuotaSettings)

	// Convert customfields
	dst.CustomFields = make(map[string]ConfigCustomField, len(cc.CustomFields))
	for key, val := range cc.CustomFields {
		// Use the method defined on ConfigCustomField
		dst.CustomFields[key] = convertFromCustomFields(val)
	}

	return dst
}

// convertFromQuotaSettings converts a v1alpha2.ConfigQuotaSettings to ConfigQuotaSettings.
func convertFromQuotaSettings(cqs v1alpha2.ConfigQuotaSettings) ConfigQuotaSettings {
	return ConfigQuotaSettings{
		Clusterwide: cqs.Clusterwide,
		Ratio:       cqs.Ratio,
		DefQuota:    cqs.DefQuota,
		MinQuotas:   cqs.MinQuotas,
		MaxQuotas:   cqs.MaxQuotas,
	}
}

// convertFromCustomFields converts a v1alpha2.ConfigCustomField to ConfigCustomField.
func convertFromCustomFields(ccf v1alpha2.ConfigCustomField) ConfigCustomField {
	return ConfigCustomField{
		Validation: ccf.Validation,
		Default:    ccf.Default,
		Template:   ccf.Template,
		Required:   ccf.Required,
	}
}

// convertFromRoleMappings converts a v1alpha2.ConfigRoleMappings to ConfigRoleMappings.
func convertFromRoleMappings(crm v1alpha2.ConfigRoleMappings) ConfigRoleMappings {
	if crm == nil {
		return nil
	}

	out := make(ConfigRoleMappings, len(crm))
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

// convertFromValidations converts a v1alpha2.PaasConfigValidations to PaasConfigValidations.
func convertFromValidations(pcv v1alpha2.PaasConfigValidations) PaasConfigValidations {
	if pcv == nil {
		return nil
	}

	out := make(PaasConfigValidations, len(pcv))
	for key, inner := range pcv {
		if inner == nil {
			out[key] = nil
			continue
		}
		copied := make(PaasConfigTypeValidations, len(inner))
		maps.Copy(copied, inner)
		out[key] = copied
	}
	return out
}

// ---------- convertTo

// ConvertTo converts this PaasConfig (v1alpha1) to the Hub version (v1alpha2).
func (pc *PaasConfig) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*v1alpha2.PaasConfig)
	if !ok {
		return fmt.Errorf("cannot convert from %T: expected PaasConfig", dstRaw)
	}

	logger := log.With().
		Any("conversion", dst.GetObjectKind().GroupVersionKind()).
		Str("name", dst.GetName()).
		Logger()
	logger.Debug().Msg("Starting conversion from spoke (v1alpha1) to hub (v1alpha2)")

	dst.ObjectMeta = pc.ObjectMeta
	dst.Status.Conditions = pc.Status.Conditions
	dst.Spec = v1alpha2.PaasConfigSpec{}
	spec := &dst.Spec

	spec.DecryptKeysSecret = NamespacedName{}.convertTo(pc.Spec.DecryptKeysSecret)
	spec.Debug = pc.Spec.Debug

	spec.Capabilities = make(map[string]v1alpha2.ConfigCapability, len(pc.Spec.Capabilities))
	for key, val := range pc.Spec.Capabilities {
		spec.Capabilities[key] = ConfigCapability{}.convertTo(val)
	}

	spec.ClusterWideArgoCDNamespace = pc.Spec.ClusterWideArgoCDNamespace
	spec.QuotaLabel = pc.Spec.QuotaLabel
	spec.RequestorLabel = pc.Spec.RequestorLabel
	spec.ManagedByLabel = pc.Spec.ManagedByLabel
	spec.ManagedBySuffix = pc.Spec.ManagedBySuffix
	spec.RoleMappings = ConfigRoleMappings{}.convertTo(pc.Spec.RoleMappings)
	spec.Validations = PaasConfigValidations{}.convertTo(pc.Spec.Validations)

	return nil
}

// convertTo converts the given NamespacedName to v1alpha2.NamespacedName
func (nn NamespacedName) convertTo(n NamespacedName) v1alpha2.NamespacedName {
	return v1alpha2.NamespacedName{
		Namespace: n.Namespace,
		Name:      n.Name,
	}
}

func (cc ConfigCapability) convertTo(src ConfigCapability) v1alpha2.ConfigCapability {
	dst := v1alpha2.ConfigCapability{}
	dst.AppSet = src.AppSet
	dst.DefaultPermissions = v1alpha2.ConfigCapPerm(src.DefaultPermissions)
	dst.ExtraPermissions = v1alpha2.ConfigCapPerm(src.ExtraPermissions)
	dst.QuotaSettings = ConfigQuotaSettings{}.convertTo(src.QuotaSettings)

	dst.CustomFields = make(map[string]v1alpha2.ConfigCustomField, len(src.CustomFields))
	for key, val := range src.CustomFields {
		dst.CustomFields[key] = ConfigCustomField{}.convertTo(val)
	}
	return dst
}

func (cqs ConfigQuotaSettings) convertTo(src ConfigQuotaSettings) v1alpha2.ConfigQuotaSettings {
	return v1alpha2.ConfigQuotaSettings{
		Clusterwide: src.Clusterwide,
		Ratio:       src.Ratio,
		DefQuota:    src.DefQuota,
		MinQuotas:   src.MinQuotas,
		MaxQuotas:   src.MaxQuotas,
	}
}

func (ccf ConfigCustomField) convertTo(src ConfigCustomField) v1alpha2.ConfigCustomField {
	return v1alpha2.ConfigCustomField{
		Validation: src.Validation,
		Default:    src.Default,
		Template:   src.Template,
		Required:   src.Required,
	}
}

func (crm ConfigRoleMappings) convertTo(src ConfigRoleMappings) v1alpha2.ConfigRoleMappings {
	if src == nil {
		return nil
	}

	dst := make(v1alpha2.ConfigRoleMappings, len(src))
	for k, v := range src {
		dst[k] = append([]string(nil), v...)
	}
	return dst
}

func (pcv PaasConfigValidations) convertTo(src PaasConfigValidations) v1alpha2.PaasConfigValidations {
	if src == nil {
		return nil
	}

	dst := make(v1alpha2.PaasConfigValidations, len(src))
	for key, inner := range src {
		if inner == nil {
			dst[key] = nil
			continue
		}
		copied := make(v1alpha2.PaasConfigTypeValidations, len(inner))
		maps.Copy(copied, inner)
		dst[key] = copied
	}
	return dst
}
