/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha2

import (
	"testing"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var paasconfigExV1Alpha1 = &v1alpha1.PaasConfig{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "config-example",
		Namespace: "platform",
	},
	Spec: v1alpha1.PaasConfigSpec{
		Debug: true,
		Capabilities: map[string]v1alpha1.ConfigCapability{
			"example": {
				AppSet: "set",
				DefaultPermissions: v1alpha1.ConfigCapPerm{
					"read":  {"value1", "value2"},
					"write": {},
				},
				ExtraPermissions: v1alpha1.ConfigCapPerm{
					"write": {},
				},
				QuotaSettings: v1alpha1.ConfigQuotaSettings{
					Clusterwide: true,
					Ratio:       0.75,
					DefQuota: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
				},
				CustomFields: map[string]v1alpha1.ConfigCustomField{
					"env": {
						Validation: "string",
						Default:    "dev",
						Template:   "input",
						Required:   true,
					},
				},
			},
		},
		RoleMappings: v1alpha1.ConfigRoleMappings{
			"admin": {"alice", "bob"},
		},
		Validations: v1alpha1.PaasConfigValidations{
			"env": {
				"type": "string",
			},
		},
	},
	Status: v1alpha1.PaasConfigStatus{
		Conditions: []metav1.Condition{
			{
				Type:   "Ready",
				Status: metav1.ConditionTrue,
			},
		},
	},
}

var paasconfigExV1Alpha2 = &PaasConfig{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "config-example",
		Namespace: "platform",
	},
	Spec: PaasConfigSpec{
		Debug: true,
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
					DefQuota: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
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

func TestConvertPaasConfigToV1alpha2Fromv1alpha1(t *testing.T) {
	src := paasconfigExV1Alpha2.DeepCopy()
	dst := &v1alpha1.PaasConfig{}

	err := src.ConvertTo(dst)

	assert.NoError(t, err)
	assert.Equal(t, paasconfigExV1Alpha1, dst)
}

func TestConvertPaasConfigFromV1alpha1ToV1alpha2(t *testing.T) {
	src := paasconfigExV1Alpha1.DeepCopy()
	dst := &PaasConfig{}

	err := dst.ConvertFrom(src)

	assert.NoError(t, err)
	assert.Equal(t, paasconfigExV1Alpha2, dst)
}
