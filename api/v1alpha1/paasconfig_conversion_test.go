/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"testing"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	paasquota "github.com/belastingdienst/opr-paas/v3/pkg/quota"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var paasconfigExV1Alpha1 = &PaasConfig{
	ObjectMeta: metav1.ObjectMeta{
		Name: "config-example",
	},
	Spec: PaasConfigSpec{
		Debug: true,
		ComponentsDebug: map[string]bool{
			"component1": true,
			"component2": false,
		},
		Capabilities: map[string]ConfigCapability{
			"example": {
				AppSet: "set",
				DefaultPermissions: ConfigCapPerm{
					"read":  {"value1", "value2"},
					"write": {},
				},
				ExtraPermissions: ConfigCapPerm{
					"write": {},
				},
				QuotaSettings: ConfigQuotaSettings{
					Clusterwide: true,
					Ratio:       0.75,
					DefQuota: paasquota.Quota{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
					MinQuotas: paasquota.Quota{},
					MaxQuotas: paasquota.Quota{},
				},
				CustomFields: map[string]ConfigCustomField{
					"env": {
						Validation: "string",
						Default:    "dev",
						Template:   "input",
						Required:   true,
					},
				},
			},
		},
		RoleMappings: ConfigRoleMappings{
			"admin": {"alice", "bob"},
		},
		Validations: PaasConfigValidations{
			"env": {
				"type": "string",
			},
		},
	},
	Status: PaasConfigStatus{
		Conditions: []metav1.Condition{
			{
				Type:   "Ready",
				Status: metav1.ConditionTrue,
			},
		},
	},
}

var paasconfigExV1Alpha2 = &v1alpha2.PaasConfig{
	ObjectMeta: metav1.ObjectMeta{
		Name: "config-example",
	},
	Spec: v1alpha2.PaasConfigSpec{
		Debug: true,
		ComponentsDebug: map[string]bool{
			"component1": true,
			"component2": false,
		},
		Capabilities: map[string]v1alpha2.ConfigCapability{
			"example": {
				AppSet: "set",
				DefaultPermissions: v1alpha2.ConfigCapPerm{
					"read":  {"value1", "value2"},
					"write": {},
				},
				ExtraPermissions: v1alpha2.ConfigCapPerm{
					"write": {},
				},
				QuotaSettings: v1alpha2.ConfigQuotaSettings{
					Clusterwide: true,
					Ratio:       0.75,
					DefQuota: paasquota.Quota{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
					MinQuotas: paasquota.Quota{},
					MaxQuotas: paasquota.Quota{},
				},
				CustomFields: map[string]v1alpha2.ConfigCustomField{
					"env": {
						Validation: "string",
						Default:    "dev",
						Template:   "input",
						Required:   true,
					},
				},
			},
		},
		RoleMappings: v1alpha2.ConfigRoleMappings{
			"admin": {"alice", "bob"},
		},
		Validations: v1alpha2.PaasConfigValidations{
			"env": {
				"type": "string",
			},
		},
	},
	Status: v1alpha2.PaasConfigStatus{
		Conditions: []metav1.Condition{
			{
				Type:   "Ready",
				Status: metav1.ConditionTrue,
			},
		},
	},
}

func TestConvertPaasConfigToV1alpha2Fromv1alpha1(t *testing.T) {
	src := paasconfigExV1Alpha2.DeepCopy()
	dst := &PaasConfig{}

	err := dst.ConvertFrom(src)

	assert.NoError(t, err)
	assert.Equal(t, paasconfigExV1Alpha1, dst)
}

func TestConvertPaasConfigFromV1alpha1ToV1alpha2(t *testing.T) {
	src := paasconfigExV1Alpha1.DeepCopy()
	dst := &v1alpha2.PaasConfig{}

	err := src.ConvertTo(dst)

	assert.NoError(t, err)
	assert.Equal(t, paasconfigExV1Alpha2, dst)
}
