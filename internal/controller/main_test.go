/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"testing"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/stretchr/testify/assert"
)

func TestMain_intersection(t *testing.T) {
	l1 := []string{"v1", "v2", "v2", "v3", "v4"}
	l2 := []string{"v2", "v2", "v3", "v5"}
	li := intersect(l1, l2)
	// Expected to have only all values that exist in list 1 and 2, only once (unique)
	lExpected := []string{"v2", "v3"}
	assert.ElementsMatch(t, li, lExpected, "result of intersection not as expected")
}

func TestMergeSecrets(t *testing.T) {
	tests := []struct {
		name     string
		base     map[string]string
		override map[string]string
		want     map[string]string
	}{
		{
			name:     "empty base and override",
			base:     map[string]string{},
			override: map[string]string{},
			want:     map[string]string{},
		},
		{
			name:     "base only",
			base:     map[string]string{"a1": "1"},
			override: map[string]string{},
			want:     map[string]string{"a1": "1"},
		},
		{
			name:     "override only",
			base:     map[string]string{},
			override: map[string]string{"b": "b2"},
			want:     map[string]string{"b": "b2"},
		},
		{
			name:     "override replaces base",
			base:     map[string]string{"c": "c1"},
			override: map[string]string{"c": "c2"},
			want:     map[string]string{"c": "c2"},
		},
		{
			name:     "override adds to base",
			base:     map[string]string{"a": "1"},
			override: map[string]string{"b": "2"},
			want:     map[string]string{"a": "1", "b": "2"},
		},
		{
			name:     "multiple overrides",
			base:     map[string]string{"f": "1", "c": "3"},
			override: map[string]string{"f": "10", "g": "20"},
			want:     map[string]string{"f": "10", "g": "20", "c": "3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// copy maps to avoid mutating original test cases
			baseCopy := maps.Clone(tt.base)
			overrideCopy := maps.Clone(tt.override)

			got := mergeSecrets(baseCopy, overrideCopy)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeSecrets() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
				ctx: context.WithValue(context.Background(), contextKeyPaasConfig, v1alpha2.PaasConfig{
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
				ctx: context.WithValue(context.Background(), contextKeyPaasConfig, "not-a-config"),
			},
			want:    v1alpha2.PaasConfig{},
			wantErr: assert.Error,
		},
		{
			name: "config fails in context as pointer",
			args: args{
				ctx: context.WithValue(context.Background(), contextKeyPaasConfig, &v1alpha2.PaasConfig{
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
			got, err := getConfigFromContext(tt.args.ctx)
			if !tt.wantErr(t, err, fmt.Sprintf("getConfigFromContext(%v)", tt.args.ctx)) {
				return
			}
			assert.Equalf(t, tt.want, got, "getConfigFromContext(%v)", tt.args.ctx)
		})
	}
}
