package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestPaasConfig_IsActive(t *testing.T) {
	type fields struct {
		Status PaasConfigStatus
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "No status.conditions, expect false",
			fields: fields{
				Status: PaasConfigStatus{
					Conditions: nil,
				},
			},
			want: false,
		},
		{
			name: "Condition TypeActivePaasConfig present but status unknown, expect false",
			fields: fields{
				Status: PaasConfigStatus{
					Conditions: []metav1.Condition{
						{
							Type:   TypeActivePaasConfig,
							Status: metav1.ConditionUnknown,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "Condition TypeActivePaasConfig present and status true, expect true",
			fields: fields{
				Status: PaasConfigStatus{
					Conditions: []metav1.Condition{
						{
							Type:   TypeActivePaasConfig,
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := PaasConfig{
				Status: tt.fields.Status,
			}
			assert.Equalf(t, tt.want, p.isActive(), "IsActive()")
		})
	}
}

func TestConfigCapPerm_Roles(t *testing.T) {
	t.Run("Empty map returns empty slice", func(t *testing.T) {
		ccp := ConfigCapPerm{}
		result := ccp.Roles()
		assert.Empty(t, result)
	})

	t.Run("Single permission with roles", func(t *testing.T) {
		ccp := ConfigCapPerm{
			"read": {"user"},
		}
		result := ccp.Roles()
		assert.Equal(t, []string{"user"}, result)
	})

	t.Run("Multiple permissions with multiple roles", func(t *testing.T) {
		ccp := ConfigCapPerm{
			"read":  {"user", "guest"},
			"write": {"admin"},
		}
		result := ccp.Roles()
		assert.ElementsMatch(t, []string{"user", "guest", "admin"}, result)
	})

	t.Run("Handles empty role lists", func(t *testing.T) {
		ccp := ConfigCapPerm{
			"read":  {},
			"write": {"admin"},
		}
		result := ccp.Roles()
		assert.Equal(t, []string{"admin"}, result)
	})
}

func TestConfigCapPerm_ServiceAccounts(t *testing.T) {
	t.Run("Empty map returns empty slice", func(t *testing.T) {
		ccp := ConfigCapPerm{}
		result := ccp.ServiceAccounts()
		assert.Empty(t, result)
	})

	t.Run("Single service account", func(t *testing.T) {
		ccp := ConfigCapPerm{
			"sa-reader": {"read"},
		}
		result := ccp.ServiceAccounts()
		assert.Equal(t, []string{"sa-reader"}, result)
	})

	t.Run("Multiple service accounts", func(t *testing.T) {
		ccp := ConfigCapPerm{
			"sa-reader": {"read"},
			"sa-writer": {"write"},
			"sa-admin":  {"admin"},
		}
		result := ccp.ServiceAccounts()
		assert.ElementsMatch(t, []string{"sa-reader", "sa-writer", "sa-admin"}, result)
	})
}

func TestConfigRolesSas_Merge(t *testing.T) {
	t.Run("Merge non-overlapping roles", func(t *testing.T) {
		base := configRolesSas{
			"reader": {"sa1": true},
		}
		other := configRolesSas{
			"writer": {"sa2": true},
		}

		result := base.Merge(other)

		expected := configRolesSas{
			"reader": {"sa1": true},
			"writer": {"sa2": true},
		}
		assert.Equal(t, expected, result)
	})

	t.Run("Merge overlapping roles, additive", func(t *testing.T) {
		base := configRolesSas{
			"admin": {"sa1": true},
		}
		other := configRolesSas{
			"admin": {"sa2": true},
		}

		result := base.Merge(other)

		expected := configRolesSas{
			"admin": {
				"sa1": true,
				"sa2": true,
			},
		}
		assert.Equal(t, expected, result)
	})

	t.Run("Merge overwrites values for existing SAs", func(t *testing.T) {
		base := configRolesSas{
			"admin": {"sa1": true},
		}
		other := configRolesSas{
			"admin": {"sa1": false},
		}

		result := base.Merge(other)

		expected := configRolesSas{
			"admin": {"sa1": false},
		}
		assert.Equal(t, expected, result)
	})
}

func TestActivePaasConfigUpdated_UpdateEvents(t *testing.T) {
	pred := ActivePaasConfigUpdated()

	trueCondition := metav1.Condition{
		Type:               TypeActivePaasConfig,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Time{Time: time.Now()},
	}

	t.Run("returns true when Active condition transitions to true", func(t *testing.T) {
		oldObj := &PaasConfig{}
		newObj := &PaasConfig{
			Status: PaasConfigStatus{
				Conditions: []metav1.Condition{trueCondition},
			},
		}
		result := pred.Update(event.UpdateEvent{
			ObjectOld: oldObj,
			ObjectNew: newObj,
		})
		assert.True(t, result)
	})

	t.Run("returns true when Active is true and spec has changed", func(t *testing.T) {
		oldObj := &PaasConfig{
			Spec: PaasConfigSpec{RequestorLabel: "a"},
			Status: PaasConfigStatus{
				Conditions: []metav1.Condition{trueCondition},
			},
		}
		newObj := &PaasConfig{
			Spec: PaasConfigSpec{RequestorLabel: "b"},
			Status: PaasConfigStatus{
				Conditions: []metav1.Condition{trueCondition},
			},
		}
		result := pred.Update(event.UpdateEvent{
			ObjectOld: oldObj,
			ObjectNew: newObj,
		})
		assert.True(t, result)
	})

	t.Run("returns false when Active is true but spec is unchanged", func(t *testing.T) {
		oldObj := &PaasConfig{
			Spec: PaasConfigSpec{RequestorLabel: "same"},
			Status: PaasConfigStatus{
				Conditions: []metav1.Condition{trueCondition},
			},
		}
		newObj := &PaasConfig{
			Spec: PaasConfigSpec{RequestorLabel: "same"},
			Status: PaasConfigStatus{
				Conditions: []metav1.Condition{trueCondition},
			},
		}
		result := pred.Update(event.UpdateEvent{
			ObjectOld: oldObj,
			ObjectNew: newObj,
		})
		assert.False(t, result)
	})

	t.Run("returns false when Active is not true", func(t *testing.T) {
		oldObj := &PaasConfig{}
		newObj := &PaasConfig{}
		result := pred.Update(event.UpdateEvent{
			ObjectOld: oldObj,
			ObjectNew: newObj,
		})
		assert.False(t, result)
	})
}

func TestActivePaasConfigUpdated_NonUpdateEvents(t *testing.T) {
	pred := ActivePaasConfigUpdated()

	t.Run("CreateFunc returns false", func(t *testing.T) {
		assert.False(t, pred.Create(event.CreateEvent{}))
	})

	t.Run("DeleteFunc returns false", func(t *testing.T) {
		assert.False(t, pred.Delete(event.DeleteEvent{}))
	})

	t.Run("GenericFunc returns false", func(t *testing.T) {
		assert.False(t, pred.Generic(event.GenericEvent{}))
	})
}
