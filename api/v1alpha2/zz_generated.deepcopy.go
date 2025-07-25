//go:build !ignore_autogenerated

/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in ConfigCapPerm) DeepCopyInto(out *ConfigCapPerm) {
	{
		in := &in
		*out = make(ConfigCapPerm, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = make([]string, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigCapPerm.
func (in ConfigCapPerm) DeepCopy() ConfigCapPerm {
	if in == nil {
		return nil
	}
	out := new(ConfigCapPerm)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in ConfigCapabilities) DeepCopyInto(out *ConfigCapabilities) {
	{
		in := &in
		*out = make(ConfigCapabilities, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigCapabilities.
func (in ConfigCapabilities) DeepCopy() ConfigCapabilities {
	if in == nil {
		return nil
	}
	out := new(ConfigCapabilities)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigCapability) DeepCopyInto(out *ConfigCapability) {
	*out = *in
	in.QuotaSettings.DeepCopyInto(&out.QuotaSettings)
	if in.ExtraPermissions != nil {
		in, out := &in.ExtraPermissions, &out.ExtraPermissions
		*out = make(ConfigCapPerm, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = make([]string, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
	if in.DefaultPermissions != nil {
		in, out := &in.DefaultPermissions, &out.DefaultPermissions
		*out = make(ConfigCapPerm, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = make([]string, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
	if in.CustomFields != nil {
		in, out := &in.CustomFields, &out.CustomFields
		*out = make(map[string]ConfigCustomField, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigCapability.
func (in *ConfigCapability) DeepCopy() *ConfigCapability {
	if in == nil {
		return nil
	}
	out := new(ConfigCapability)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigCustomField) DeepCopyInto(out *ConfigCustomField) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigCustomField.
func (in *ConfigCustomField) DeepCopy() *ConfigCustomField {
	if in == nil {
		return nil
	}
	out := new(ConfigCustomField)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigFeatureFlags) DeepCopyInto(out *ConfigFeatureFlags) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigFeatureFlags.
func (in *ConfigFeatureFlags) DeepCopy() *ConfigFeatureFlags {
	if in == nil {
		return nil
	}
	out := new(ConfigFeatureFlags)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigLdap) DeepCopyInto(out *ConfigLdap) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigLdap.
func (in *ConfigLdap) DeepCopy() *ConfigLdap {
	if in == nil {
		return nil
	}
	out := new(ConfigLdap)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigQuotaSettings) DeepCopyInto(out *ConfigQuotaSettings) {
	*out = *in
	if in.DefQuota != nil {
		in, out := &in.DefQuota, &out.DefQuota
		*out = make(map[corev1.ResourceName]resource.Quantity, len(*in))
		for key, val := range *in {
			(*out)[key] = val.DeepCopy()
		}
	}
	if in.MinQuotas != nil {
		in, out := &in.MinQuotas, &out.MinQuotas
		*out = make(map[corev1.ResourceName]resource.Quantity, len(*in))
		for key, val := range *in {
			(*out)[key] = val.DeepCopy()
		}
	}
	if in.MaxQuotas != nil {
		in, out := &in.MaxQuotas, &out.MaxQuotas
		*out = make(map[corev1.ResourceName]resource.Quantity, len(*in))
		for key, val := range *in {
			(*out)[key] = val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigQuotaSettings.
func (in *ConfigQuotaSettings) DeepCopy() *ConfigQuotaSettings {
	if in == nil {
		return nil
	}
	out := new(ConfigQuotaSettings)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in ConfigRoleMappings) DeepCopyInto(out *ConfigRoleMappings) {
	{
		in := &in
		*out = make(ConfigRoleMappings, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = make([]string, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigRoleMappings.
func (in ConfigRoleMappings) DeepCopy() ConfigRoleMappings {
	if in == nil {
		return nil
	}
	out := new(ConfigRoleMappings)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in ConfigRolesSas) DeepCopyInto(out *ConfigRolesSas) {
	{
		in := &in
		*out = make(ConfigRolesSas, len(*in))
		for key, val := range *in {
			var outVal map[string]bool
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = make(map[string]bool, len(*in))
				for key, val := range *in {
					(*out)[key] = val
				}
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigRolesSas.
func (in ConfigRolesSas) DeepCopy() ConfigRolesSas {
	if in == nil {
		return nil
	}
	out := new(ConfigRolesSas)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in ConfigTemplatingItem) DeepCopyInto(out *ConfigTemplatingItem) {
	{
		in := &in
		*out = make(ConfigTemplatingItem, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigTemplatingItem.
func (in ConfigTemplatingItem) DeepCopy() ConfigTemplatingItem {
	if in == nil {
		return nil
	}
	out := new(ConfigTemplatingItem)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigTemplatingItems) DeepCopyInto(out *ConfigTemplatingItems) {
	*out = *in
	if in.GenericCapabilityFields != nil {
		in, out := &in.GenericCapabilityFields, &out.GenericCapabilityFields
		*out = make(ConfigTemplatingItem, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.ClusterQuotaLabels != nil {
		in, out := &in.ClusterQuotaLabels, &out.ClusterQuotaLabels
		*out = make(ConfigTemplatingItem, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.GroupLabels != nil {
		in, out := &in.GroupLabels, &out.GroupLabels
		*out = make(ConfigTemplatingItem, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.NamespaceLabels != nil {
		in, out := &in.NamespaceLabels, &out.NamespaceLabels
		*out = make(ConfigTemplatingItem, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.RoleBindingLabels != nil {
		in, out := &in.RoleBindingLabels, &out.RoleBindingLabels
		*out = make(ConfigTemplatingItem, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigTemplatingItems.
func (in *ConfigTemplatingItems) DeepCopy() *ConfigTemplatingItems {
	if in == nil {
		return nil
	}
	out := new(ConfigTemplatingItems)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NamespacedName) DeepCopyInto(out *NamespacedName) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NamespacedName.
func (in *NamespacedName) DeepCopy() *NamespacedName {
	if in == nil {
		return nil
	}
	out := new(NamespacedName)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Paas) DeepCopyInto(out *Paas) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Paas.
func (in *Paas) DeepCopy() *Paas {
	if in == nil {
		return nil
	}
	out := new(Paas)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Paas) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in PaasCapabilities) DeepCopyInto(out *PaasCapabilities) {
	{
		in := &in
		*out = make(PaasCapabilities, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PaasCapabilities.
func (in PaasCapabilities) DeepCopy() PaasCapabilities {
	if in == nil {
		return nil
	}
	out := new(PaasCapabilities)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PaasCapability) DeepCopyInto(out *PaasCapability) {
	*out = *in
	if in.CustomFields != nil {
		in, out := &in.CustomFields, &out.CustomFields
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	out.Quota = in.Quota.DeepCopy()
	if in.Secrets != nil {
		in, out := &in.Secrets, &out.Secrets
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PaasCapability.
func (in *PaasCapability) DeepCopy() *PaasCapability {
	if in == nil {
		return nil
	}
	out := new(PaasCapability)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PaasConfig) DeepCopyInto(out *PaasConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PaasConfig.
func (in *PaasConfig) DeepCopy() *PaasConfig {
	if in == nil {
		return nil
	}
	out := new(PaasConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PaasConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PaasConfigList) DeepCopyInto(out *PaasConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PaasConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PaasConfigList.
func (in *PaasConfigList) DeepCopy() *PaasConfigList {
	if in == nil {
		return nil
	}
	out := new(PaasConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PaasConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PaasConfigSpec) DeepCopyInto(out *PaasConfigSpec) {
	*out = *in
	out.DecryptKeysSecret = in.DecryptKeysSecret
	if in.Capabilities != nil {
		in, out := &in.Capabilities, &out.Capabilities
		*out = make(ConfigCapabilities, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.RoleMappings != nil {
		in, out := &in.RoleMappings, &out.RoleMappings
		*out = make(ConfigRoleMappings, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = make([]string, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
	out.FeatureFlags = in.FeatureFlags
	if in.Validations != nil {
		in, out := &in.Validations, &out.Validations
		*out = make(PaasConfigValidations, len(*in))
		for key, val := range *in {
			var outVal map[string]string
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = make(PaasConfigTypeValidations, len(*in))
				for key, val := range *in {
					(*out)[key] = val
				}
			}
			(*out)[key] = outVal
		}
	}
	in.Templating.DeepCopyInto(&out.Templating)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PaasConfigSpec.
func (in *PaasConfigSpec) DeepCopy() *PaasConfigSpec {
	if in == nil {
		return nil
	}
	out := new(PaasConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PaasConfigStatus) DeepCopyInto(out *PaasConfigStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PaasConfigStatus.
func (in *PaasConfigStatus) DeepCopy() *PaasConfigStatus {
	if in == nil {
		return nil
	}
	out := new(PaasConfigStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in PaasConfigTypeValidations) DeepCopyInto(out *PaasConfigTypeValidations) {
	{
		in := &in
		*out = make(PaasConfigTypeValidations, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PaasConfigTypeValidations.
func (in PaasConfigTypeValidations) DeepCopy() PaasConfigTypeValidations {
	if in == nil {
		return nil
	}
	out := new(PaasConfigTypeValidations)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in PaasConfigValidations) DeepCopyInto(out *PaasConfigValidations) {
	{
		in := &in
		*out = make(PaasConfigValidations, len(*in))
		for key, val := range *in {
			var outVal map[string]string
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = make(PaasConfigTypeValidations, len(*in))
				for key, val := range *in {
					(*out)[key] = val
				}
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PaasConfigValidations.
func (in PaasConfigValidations) DeepCopy() PaasConfigValidations {
	if in == nil {
		return nil
	}
	out := new(PaasConfigValidations)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PaasGroup) DeepCopyInto(out *PaasGroup) {
	*out = *in
	if in.Users != nil {
		in, out := &in.Users, &out.Users
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Roles != nil {
		in, out := &in.Roles, &out.Roles
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PaasGroup.
func (in *PaasGroup) DeepCopy() *PaasGroup {
	if in == nil {
		return nil
	}
	out := new(PaasGroup)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in PaasGroups) DeepCopyInto(out *PaasGroups) {
	{
		in := &in
		*out = make(PaasGroups, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PaasGroups.
func (in PaasGroups) DeepCopy() PaasGroups {
	if in == nil {
		return nil
	}
	out := new(PaasGroups)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PaasList) DeepCopyInto(out *PaasList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Paas, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PaasList.
func (in *PaasList) DeepCopy() *PaasList {
	if in == nil {
		return nil
	}
	out := new(PaasList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PaasList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PaasNS) DeepCopyInto(out *PaasNS) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PaasNS.
func (in *PaasNS) DeepCopy() *PaasNS {
	if in == nil {
		return nil
	}
	out := new(PaasNS)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PaasNS) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PaasNSList) DeepCopyInto(out *PaasNSList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PaasNS, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PaasNSList.
func (in *PaasNSList) DeepCopy() *PaasNSList {
	if in == nil {
		return nil
	}
	out := new(PaasNSList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PaasNSList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PaasNSSpec) DeepCopyInto(out *PaasNSSpec) {
	*out = *in
	if in.Groups != nil {
		in, out := &in.Groups, &out.Groups
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Secrets != nil {
		in, out := &in.Secrets, &out.Secrets
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PaasNSSpec.
func (in *PaasNSSpec) DeepCopy() *PaasNSSpec {
	if in == nil {
		return nil
	}
	out := new(PaasNSSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PaasNamespace) DeepCopyInto(out *PaasNamespace) {
	*out = *in
	if in.Groups != nil {
		in, out := &in.Groups, &out.Groups
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Secrets != nil {
		in, out := &in.Secrets, &out.Secrets
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PaasNamespace.
func (in *PaasNamespace) DeepCopy() *PaasNamespace {
	if in == nil {
		return nil
	}
	out := new(PaasNamespace)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in PaasNamespaces) DeepCopyInto(out *PaasNamespaces) {
	{
		in := &in
		*out = make(PaasNamespaces, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PaasNamespaces.
func (in PaasNamespaces) DeepCopy() PaasNamespaces {
	if in == nil {
		return nil
	}
	out := new(PaasNamespaces)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PaasSpec) DeepCopyInto(out *PaasSpec) {
	*out = *in
	out.Quota = in.Quota.DeepCopy()
	if in.Capabilities != nil {
		in, out := &in.Capabilities, &out.Capabilities
		*out = make(PaasCapabilities, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.Groups != nil {
		in, out := &in.Groups, &out.Groups
		*out = make(PaasGroups, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.Namespaces != nil {
		in, out := &in.Namespaces, &out.Namespaces
		*out = make(PaasNamespaces, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.Secrets != nil {
		in, out := &in.Secrets, &out.Secrets
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PaasSpec.
func (in *PaasSpec) DeepCopy() *PaasSpec {
	if in == nil {
		return nil
	}
	out := new(PaasSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PaasStatus) DeepCopyInto(out *PaasStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PaasStatus.
func (in *PaasStatus) DeepCopy() *PaasStatus {
	if in == nil {
		return nil
	}
	out := new(PaasStatus)
	in.DeepCopyInto(out)
	return out
}
