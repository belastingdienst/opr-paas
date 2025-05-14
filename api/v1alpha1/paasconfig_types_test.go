package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			assert.Equalf(t, tt.want, p.IsActive(), "IsActive()")
		})
	}
}
