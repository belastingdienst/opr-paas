/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

const (
	// Already declared in paas_types_test.go
	// paasName   = "paasName"
	paasNsName    = "paasns"
	excludedLabel = "app.kubernetes.io/instance"
)

// PaasNS

func TestPaasNS_NamespaceName(t *testing.T) {
	// subtest: valid paas and paasns names
	pns := PaasNS{ObjectMeta: metav1.ObjectMeta{Name: paasNsName}, Spec: PaasNSSpec{Paas: paasName}}
	output := pns.NamespaceName()
	assert.Equal(t, join(paasName, paasNsName), output)

	// subtest: empty paas and/or paasns names
	pns = PaasNS{ObjectMeta: metav1.ObjectMeta{Name: ""}, Spec: PaasNSSpec{Paas: paasName}}
	assert.PanicsWithError(
		t,
		"invalid paas or paasns name (empty)",
		func() { pns.NamespaceName() },
		"Should panic if PaasNS name is empty string",
	)

	pns = PaasNS{ObjectMeta: metav1.ObjectMeta{Name: paasNsName}, Spec: PaasNSSpec{Paas: ""}}
	assert.PanicsWithError(
		t,
		"invalid paas or paasns name (empty)",
		func() { pns.NamespaceName() },
		"Should panic if Paas name is empty string",
	)
}

// PaasNsStatus

func TestPaasNsStatus_Truncate(t *testing.T) {
	ps := PaasNsStatus{
		Messages: []string{
			"Message 1",
			"Message 2",
		},
	}

	assert.NotNil(t, ps.Messages)
	assert.Len(t, ps.Messages, 2)

	ps.truncate()
	assert.NotNil(t, ps.Messages)
	assert.Empty(t, ps.Messages)
}
