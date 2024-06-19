/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"testing"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

func TestReadPaasFile(t *testing.T) {
	expectedPaas := &v1alpha1.Paas{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Paas",
			APIVersion: "cpet.belastingdienst.nl/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "tst-tst",
		},
		Spec:   v1alpha1.PaasSpec{},
		Status: v1alpha1.PaasStatus{},
	}

	// invalid path
	paas, typeString, err := readPaasFile("invalid/path")
	expectedErrorMsg := "open invalid/path: no such file or directory"
	assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
	assert.Nil(t, paas)
	assert.Equal(t, "unable to read paas configuration file", typeString)

	// empty yaml file
	paas, typeString, err = readPaasFile("testdata/emptyPaas.yml")
	expectedErrorMsg = "empty paas configuration file"
	assert.NotNil(t, err)
	assert.Nil(t, paas)
	assert.Empty(t, typeString)
	assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)

	// empty json file
	paas, typeString, err = readPaasFile("testdata/emptyPaas.json")
	expectedErrorMsg = "empty paas configuration file"
	assert.NotNil(t, err)
	assert.Nil(t, paas)
	assert.Empty(t, typeString)
	assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)

	// minimal yaml file
	paas, typeString, err = readPaasFile("testdata/minimalPaas.yml")
	assert.Nil(t, err)
	assert.Equal(t, expectedPaas, paas)
	assert.Equal(t, "yaml", typeString)
	assert.NotEqual(t, "json", typeString)

	// minimal json file
	paas, typeString, err = readPaasFile("testdata/minimalPaas.json")
	assert.Nil(t, err)
	assert.Equal(t, expectedPaas, paas)
	assert.Equal(t, "json", typeString)
	assert.NotEqual(t, "yaml", typeString)

	// unsupported field in yaml file
	paas, typeString, err = readPaasFile("testdata/unsupportedFieldsPaas.yml")
	assert.Nil(t, err)
	assert.Equal(t, expectedPaas, paas)
	assert.Equal(t, "yaml", typeString)
	assert.NotEqual(t, "json", typeString)

	// invalid file format
	paas, typeString, err = readPaasFile("testdata/invalidFormat.toml")
	assert.Nil(t, paas)
	assert.Empty(t, typeString)
	assert.EqualErrorf(t, err, "file 'testdata/invalidFormat.toml' is not in a supported file format", "Invalid file format should result in error")
}

func TestHashData(t *testing.T) {
	testString := "My Wonderful Test String"
	out := hashData([]byte(testString))

	assert.Equal(t, "703fe1668c39ec0fdf3c9916d526ba4461fe10fd36bac1e2a1b708eb8a593e418eb3f92dbbd2a6e3776516b0e03743a45cfd69de6a3280afaa90f43fa1918f74", out)
}
