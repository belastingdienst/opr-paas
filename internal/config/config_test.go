/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package config

import (
	"context"
	"fmt"
	"testing"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/stretchr/testify/assert"
)

func Test_getConfigFromContext(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		want    v1alpha2.PaasConfig
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "config exists in context",
			args: args{
				ctx: context.WithValue(context.Background(), ContextKeyPaasConfig, v1alpha2.PaasConfig{
					Spec: v1alpha2.PaasConfigSpec{
						Debug: true,
					},
				}),
			},
			want: v1alpha2.PaasConfig{
				Spec: v1alpha2.PaasConfigSpec{
					Debug: true,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "no config in context",
			args: args{
				ctx: context.Background(),
			},
			want:    v1alpha2.PaasConfig{},
			wantErr: assert.Error,
		},
		{
			name: "wrong type in context",
			args: args{
				ctx: context.WithValue(context.Background(), ContextKeyPaasConfig, "not-a-config"),
			},
			want:    v1alpha2.PaasConfig{},
			wantErr: assert.Error,
		},
		{
			name: "config fails in context as pointer",
			args: args{
				ctx: context.WithValue(context.Background(), ContextKeyPaasConfig, &v1alpha2.PaasConfig{
					Spec: v1alpha2.PaasConfigSpec{
						Debug: true,
					},
				}),
			},
			want:    v1alpha2.PaasConfig{},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetConfigFromContext(tt.args.ctx)
			if !tt.wantErr(t, err, fmt.Sprintf("getConfigFromContext(%v)", tt.args.ctx)) {
				return
			}
			assert.Equalf(t, tt.want, got, "getConfigFromContext(%v)", tt.args.ctx)
		})
	}
}
