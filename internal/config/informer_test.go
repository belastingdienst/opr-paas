package config

import (
	"testing"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/api/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

func TestHandleUpdate_UpdatesActiveConfig(t *testing.T) {
	oldCfg := v1alpha2.PaasConfig{
		Spec: v1alpha2.PaasConfigSpec{RequestorLabel: "old"},
	}
	newCfg := v1alpha2.PaasConfig{
		Spec: v1alpha2.PaasConfigSpec{RequestorLabel: "new"},
	}
	SetConfig(oldCfg)

	newCfg.Status.Conditions =
		[]metav1.Condition{
			{
				Type:   v1alpha2.TypeActivePaasConfig,
				Status: metav1.ConditionTrue,
			},
		}

	defer SetConfig(v1alpha2.PaasConfig{})

	updateHandler(nil, &newCfg)

	assert.Equal(t, "new", GetConfig().Spec.RequestorLabel)
}

func TestHandleUpdate_NoChangeToInactiveConfig(t *testing.T) {
	oldCfg := v1alpha2.PaasConfig{
		Spec: v1alpha2.PaasConfigSpec{RequestorLabel: "unchanged"},
	}
	newCfg := v1alpha2.PaasConfig{
		Spec: v1alpha2.PaasConfigSpec{RequestorLabel: "new"},
	}
	SetConfig(oldCfg)

	// inactive config
	newCfg.Status.Conditions = []metav1.Condition{
		{
			Type:   v1alpha1.TypeActivePaasConfig,
			Status: metav1.ConditionUnknown,
		},
	}

	defer SetConfig(v1alpha2.PaasConfig{})

	updateHandler(nil, &newCfg)

	assert.Equal(t, "unchanged", GetConfig().Spec.RequestorLabel)
}
