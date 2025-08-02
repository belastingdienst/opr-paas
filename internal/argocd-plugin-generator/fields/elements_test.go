/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package fields_test

import (
	"testing"

	"github.com/belastingdienst/opr-paas/v3/internal/argocd-plugin-generator/fields"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	key1     = "a"
	value1   = "b"
	key2     = "c"
	value2   = 6.0
	key3     = "d"
	key4     = ""
	elements = fields.Elements{
		key1: value1,
		key2: value2,
	}
)

func TestAsStringMap(t *testing.T) {
	assert.Equal(
		t,
		map[string]string{
			"a": "b",
			"c": "6",
		},
		elements.GetElementsAsStringMap(),
	)
}

func TestTryGetElementAsString(t *testing.T) {
	a, err := elements.TryGetElementAsString("a")
	require.NoError(t, err)
	assert.NotNil(t, a)
	assert.Equal(t, a, "b")
	b, err := elements.TryGetElementAsString("b")
	require.Error(t, err)
	assert.Empty(t, b)
	c, err := elements.TryGetElementAsString("c")
	require.NoError(t, err)
	assert.NotNil(t, c)
	assert.IsType(t, "6", c)
	assert.Equal(t, "6", c)
}

func TestGetElementAsString(t *testing.T) {
	require.NotNil(t, elements)
	for _, key := range []string{key1, key2, key3, key4} {
		if _, exists := elements[key]; exists {
			assert.NotEmpty(t, elements.GetElementAsString(key))
		} else {
			assert.Empty(t, elements.GetElementAsString(key))
		}
	}
}
