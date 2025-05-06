package config

import (
	"testing"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

func TestHandleUpdate_UpdatesActiveConfig(t *testing.T) {
	oldCfg := v1alpha1.PaasConfig{
		Spec: v1alpha1.PaasConfigSpec{RequestorLabel: "old"},
	}
	newCfg := v1alpha1.PaasConfig{
		Spec: v1alpha1.PaasConfigSpec{RequestorLabel: "new"},
	}
	newCfg.SetName("cfg")
	SetConfig(oldCfg)
	newCfg.Status.Conditions =
		[]metav1.Condition{
			{
				Type:   v1alpha1.TypeActivePaasConfig,
				Status: metav1.ConditionTrue,
			},
		}

	defer SetConfig(v1alpha1.PaasConfig{})

	updateHandler(nil, &newCfg)

	assert.Equal(t, "new", GetConfig().Spec.RequestorLabel)
}

func TestHandleUpdate_NoChangeToInactiveConfig(t *testing.T) {
	oldCfg := v1alpha1.PaasConfig{
		Spec: v1alpha1.PaasConfigSpec{RequestorLabel: "unchanged"},
	}
	newCfg := v1alpha1.PaasConfig{
		Spec: v1alpha1.PaasConfigSpec{RequestorLabel: "new"},
	}
	SetConfig(oldCfg)

	// inactive config
	newCfg.Status.Conditions = []metav1.Condition{
		{
			Type:   v1alpha1.TypeActivePaasConfig,
			Status: metav1.ConditionUnknown,
		},
	}

	defer SetConfig(v1alpha1.PaasConfig{})

	updateHandler(nil, &newCfg)

	assert.Equal(t, "unchanged", GetConfig().Spec.RequestorLabel)
}
