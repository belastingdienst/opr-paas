/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"testing"

	"github.com/belastingdienst/opr-paas/api/v1alpha2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var paasNsExV1Alpha1 = &PaasNS{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "foo",
		Namespace: "bar",
	},
	Spec: PaasNSSpec{
		Paas:       "",
		Groups:     []string{},
		SSHSecrets: map[string]string{},
	},
}

var paasNsExV1Alpha2 = &v1alpha2.PaasNS{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "foo",
		Namespace: "bar",
	},
	Spec: v1alpha2.PaasNSSpec{
		Paas:    "",
		Groups:  []string{},
		Secrets: map[string]string{},
	},
}

// Test conversion FROM v1alpha2 TO v1alpha1
func TestConvertPaasNsTo(t *testing.T) {
	src := paasNsExV1Alpha2.DeepCopy()
	dst := &PaasNS{}

	err := dst.ConvertFrom(src)

	assert.NoError(t, err)
	assert.Equal(t, paasNsExV1Alpha1, dst)
}

// Test conversion FROM v1alpha1 TO v1alpha2
func TestConvertPaasNsFrom(t *testing.T) {
	src := paasNsExV1Alpha1.DeepCopy()
	dst := &v1alpha2.PaasNS{}

	err := src.ConvertTo(dst)

	assert.NoError(t, err)
	assert.Equal(t, paasNsExV1Alpha2, dst)
}
