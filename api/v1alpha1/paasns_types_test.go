/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestPaasNS_ClonedLabels(t *testing.T) {
	// subtest: no labels
	pns := PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:   paasNsName,
			Labels: map[string]string{},
		},
	}

	output := pns.ClonedLabels()
	require.NotNil(t, output)
	assert.Empty(t, output)

	// subtest: single label not to be cloned
	pns.Labels[excludedLabel] = "something"
	output = pns.ClonedLabels()
	require.NotNil(t, output)
	assert.Empty(t, output)
	require.NotContains(t, output, excludedLabel)
	require.NotContains(t, output, "key1")

	// subtest: multiple labels
	for i := 0; i < 3; i++ {
		pns.Labels[fmt.Sprintf("key %d", i)] = fmt.Sprintf("key %d", i)
	}

	output = pns.ClonedLabels()
	require.NotNil(t, output)
	assert.Len(t, output, 3)
	for k := range pns.Labels {
		if k == excludedLabel {
			require.NotContains(t, output, k)
		} else {
			require.Contains(t, output, k)
		}
	}

	// subtest: single clonable label
	pns.Labels = map[string]string{"key": "something"}

	output = pns.ClonedLabels()
	require.NotNil(t, output)
	assert.Len(t, output, 1)
	require.NotContains(t, output, excludedLabel)
	require.Contains(t, output, "key")
}

func TestPaasNS_IsItMe(t *testing.T) {
	allOwners := generateReferences()
	firstOwner := allOwners[0]
	pns := PaasNS{
		TypeMeta: metav1.TypeMeta{
			Kind:       firstOwner.Kind,
			APIVersion: firstOwner.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: firstOwner.Name,
		},
	}

	for _, ref := range allOwners {
		if ref == firstOwner {
			assert.True(t, pns.IsItMe(ref))
		} else {
			assert.False(t, pns.IsItMe(ref))
		}
	}
	assert.False(t, pns.IsItMe(metav1.OwnerReference{}))
}

func TestPaasNS_AmIOwner(t *testing.T) {
	allOwners := generateReferences()
	firstOwner := allOwners[0]
	pns := PaasNS{
		TypeMeta: metav1.TypeMeta{
			Kind:       firstOwner.Kind,
			APIVersion: firstOwner.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: firstOwner.Name,
		},
	}

	someOwners := []metav1.OwnerReference{
		allOwners[0],
		allOwners[1],
	}
	noOwners := []metav1.OwnerReference{
		allOwners[2],
		allOwners[3],
	}

	empty := []metav1.OwnerReference{}

	assert.True(t, pns.AmIOwner(allOwners))
	assert.True(t, pns.AmIOwner(someOwners))
	assert.False(t, pns.AmIOwner(noOwners))
	assert.False(t, pns.AmIOwner(empty))
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

	ps.Truncate()
	assert.NotNil(t, ps.Messages)
	assert.Empty(t, ps.Messages)
}
