// Package paasresource has a Resource interface which can be used for functions that should work with multiple Paas
// resources.
package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

// Resource represents a Paas Resource (e.a. Paas, PaasNS or PaasConfig) with a `.status.conditions` slice field of
// conditions. This is a workaround to match our custom resource types; all our custom resource types have the same
// `.status.conditions` fields, but Go generics do not currently allow accessing shared struct fields via generic types.
// This is apparently a feature slated for Go 2. (https://github.com/golang/go/issues/48522#issuecomment-924380147)
type Resource interface {
	k8s.Object
	GetConditions() *[]metav1.Condition
	GetGeneration() int64
	GetName() string
}
