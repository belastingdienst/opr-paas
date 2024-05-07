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

// PaasNS

func TestPaasNS_NamespaceName(t *testing.T) {
	// subtest: valid paas and paasns names
	pns := PaasNS{ObjectMeta: metav1.ObjectMeta{Name: "paasnsname"}, Spec: PaasNSSpec{Paas: "paasname"}}
	output := pns.NamespaceName()
	assert.Equal(t, "paasname-paasnsname", output)

	// subtest: empty paas and/or paasns names
	pns = PaasNS{ObjectMeta: metav1.ObjectMeta{Name: ""}, Spec: PaasNSSpec{Paas: "paasname"}}
	assert.PanicsWithError(t, "invalid paas or paasns name", func() { pns.NamespaceName() }, "Should panic if PaasNS name is empty string")

	pns = PaasNS{ObjectMeta: metav1.ObjectMeta{Name: "paasnsname"}, Spec: PaasNSSpec{Paas: ""}}
	assert.PanicsWithError(t, "invalid paas or paasns name", func() { pns.NamespaceName() }, "Should panic if Paas name is empty string")
}

func TestPaasNS_ClonedLabels(t *testing.T) {
	// subtest: multiple labels
	pns := PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name: "paasnsname",
			Labels: map[string]string{
				"app.kubernetes.io/instance": "something",
				"key1":                       "value1",
				"key2":                       "value2",
				"key3":                       "value3",
			},
		},
	}

	output := pns.ClonedLabels()
	require.NotNil(t, output)
	assert.Len(t, output, 3)
	require.NotContains(t, output, "app.kubernetes.io/instance")
	require.Contains(t, output, "key1")
	assert.Contains(t, output, "key2")
	assert.Contains(t, output, "key3")

	// subtest: single label not to be cloned
	pns = PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name: "paasnsname",
			Labels: map[string]string{
				"app.kubernetes.io/instance": "something",
			},
		},
	}

	output = pns.ClonedLabels()
	require.NotNil(t, output)
	assert.Empty(t, output)
	require.NotContains(t, output, "app.kubernetes.io/instance")
	require.NotContains(t, output, "key1")

	// subtest: no labels
	pns = PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "paasnsname",
			Labels: map[string]string{},
		},
	}

	output = pns.ClonedLabels()
	require.NotNil(t, output)
	assert.Empty(t, output)

	// subtest: single clonable label
	pns = PaasNS{
		ObjectMeta: metav1.ObjectMeta{
			Name: "paasnsname",
			Labels: map[string]string{
				"key1": "value1",
			},
		},
	}

	output = pns.ClonedLabels()
	require.NotNil(t, output)
	assert.Len(t, output, 1)
	require.NotContains(t, output, "app.kubernetes.io/instance")
	require.Contains(t, output, "key1")
}

func TestPaasNS_IsItMe(t *testing.T) {
	pns := PaasNS{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MyKind",
			APIVersion: "1.1.1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "Some Name",
		},
	}

	test1 := metav1.OwnerReference{
		Kind:       "MyKind",
		APIVersion: "1.1.1",
		Name:       "Some Name",
	}

	test2 := metav1.OwnerReference{
		Kind:       "MyOtherKind",
		APIVersion: "1.1.1",
		Name:       "Some Name",
	}

	test3 := metav1.OwnerReference{
		Kind:       "MyKind",
		APIVersion: "1.1.0",
		Name:       "Some Name",
	}

	test4 := metav1.OwnerReference{
		Kind:       "MyKind",
		APIVersion: "1.1.1",
		Name:       "Some Other Name",
	}

	test5 := metav1.OwnerReference{
		Kind:       "MyOtherKind",
		APIVersion: "1.1.0",
		Name:       "Some Name",
	}

	test6 := metav1.OwnerReference{}

	assert.True(t, pns.IsItMe(test1))
	assert.False(t, pns.IsItMe(test2))
	assert.False(t, pns.IsItMe(test3))
	assert.False(t, pns.IsItMe(test4))
	assert.False(t, pns.IsItMe(test5))
	assert.False(t, pns.IsItMe(test6))
}

func TestPaasNS_AmIOwner(t *testing.T) {
	pns := PaasNS{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MyKind",
			APIVersion: "1.1.1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "Some Name",
		},
	}

	ref1 := metav1.OwnerReference{
		Kind:       "MyKind",
		APIVersion: "1.1.1",
		Name:       "Some Name",
	}

	ref2 := metav1.OwnerReference{
		Kind:       "MyOtherKind",
		APIVersion: "1.1.1",
		Name:       "Some Name",
	}

	owner := []metav1.OwnerReference{
		ref1,
		ref2,
	}
	notOwner := []metav1.OwnerReference{
		ref2,
		ref2,
	}

	empty := []metav1.OwnerReference{}

	assert.True(t, pns.AmIOwner(owner))
	assert.False(t, pns.AmIOwner(notOwner))
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

func TestPaasNsStatus_AddMessage(t *testing.T) {
	pns := &PaasNS{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       PaasNSSpec{},
		Status:     PaasNsStatus{},
	}
	ps := PaasNsStatus{
		Messages: []string{},
	}

	err := fmt.Errorf("Test message 1")
	ps.AddMessage(PaasStatusInfo, PaasStatusFind, pns, err.Error())

	assert.NotNil(t, ps.Messages)
	assert.Len(t, ps.Messages, 1)
	assert.Equal(t, "INFO: find for / (/, Kind=) Test message 1", ps.Messages[0])

	err = fmt.Errorf("Test message 2")
	ps.AddMessage(PaasStatusWarning, PaasStatusFind, pns, err.Error())

	assert.NotNil(t, ps.Messages)
	assert.Len(t, ps.Messages, 2)
	assert.Equal(t, "INFO: find for / (/, Kind=) Test message 1", ps.Messages[0])
	assert.Equal(t, "WARNING: find for / (/, Kind=) Test message 2", ps.Messages[1])
}

func TestPaasNsStatus_GetMessages(t *testing.T) {
	pns := &PaasNS{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       PaasNSSpec{},
		Status:     PaasNsStatus{},
	}
	ps := PaasNsStatus{
		Messages: []string{},
	}

	err := fmt.Errorf("Test message 1")
	ps.AddMessage(PaasStatusInfo, PaasStatusFind, pns, err.Error())

	assert.NotNil(t, ps.Messages)
	assert.Len(t, ps.Messages, 1)
	assert.Equal(t, "INFO: find for / (/, Kind=) Test message 1", ps.Messages[0])

	output := ps.GetMessages()
	assert.NotNil(t, output)
	assert.IsType(t, []string{}, output)
	assert.Len(t, output, 1)
	assert.Equal(t, "INFO: find for / (/, Kind=) Test message 1", output[0])
}

func TestPaasNsStatus_AddMessages(t *testing.T) {
	ps := PaasNsStatus{
		Messages: []string{
			"Message 1",
			"Message 2",
		},
	}

	msg := []string{
		"Added Message 1",
		"Added Message 2",
	}

	assert.NotNil(t, ps.Messages)
	assert.Len(t, ps.Messages, 2)
	assert.Equal(t, "Message 1", ps.Messages[0])

	ps.AddMessages(msg)

	assert.NotNil(t, ps.Messages)
	assert.IsType(t, []string{}, ps.Messages)
	assert.Len(t, ps.Messages, 4)
	assert.Equal(t, "Message 1", ps.Messages[0])
	assert.Equal(t, "Added Message 1", ps.Messages[2])
}
