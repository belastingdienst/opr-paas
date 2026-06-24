/*
Copyright 2026, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import "testing"

func TestDefaultArgocdPluginGeneratorBindAddress(t *testing.T) {
	t.Run("disabled by default", func(t *testing.T) {
		t.Setenv(argocdPluginGeneratorBindAddressEnv, "")

		if got := defaultArgocdPluginGeneratorBindAddress(); got != "0" {
			t.Fatalf("expected default bind address 0, got %q", got)
		}
	})

	t.Run("uses environment variable", func(t *testing.T) {
		t.Setenv(argocdPluginGeneratorBindAddressEnv, ":4355")

		if got := defaultArgocdPluginGeneratorBindAddress(); got != ":4355" {
			t.Fatalf("expected env bind address :4355, got %q", got)
		}
	})
}

func TestDefaultMetricsBindAddress(t *testing.T) {
	t.Run("disabled by default", func(t *testing.T) {
		t.Setenv(metricsBindAddressEnv, "")

		if got := defaultMetricsBindAddress(); got != "0" {
			t.Fatalf("expected default bind address 0, got %q", got)
		}
	})

	t.Run("uses environment variable", func(t *testing.T) {
		t.Setenv(metricsBindAddressEnv, ":8080")

		if got := defaultMetricsBindAddress(); got != ":8080" {
			t.Fatalf("expected env bind address :8080, got %q", got)
		}
	})
}

func TestDefaultMetricsSecure(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "secure by default", want: true},
		{name: "secure enabled", value: "true", want: true},
		{name: "secure disabled", value: "false", want: false},
		{name: "invalid value keeps secure default", value: "invalid", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(metricsSecureEnv, tt.value)

			if got := defaultMetricsSecure(); got != tt.want {
				t.Fatalf("expected secure metrics %t, got %t", tt.want, got)
			}
		})
	}
}
