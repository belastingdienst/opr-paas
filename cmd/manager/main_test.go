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
