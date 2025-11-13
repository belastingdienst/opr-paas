/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package fields_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/belastingdienst/opr-paas/v3/pkg/fields"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	key1     = "a"
	value1   = "b"
	key2     = "c"
	value2   = 6.0
	key3     = "d"
	value3   = map[string]string{"k1": "v1", "k2": "v2"}
	key4     = "e"
	value4   = []string{"e1", "e2"}
	key5     = "f"
	key6     = ""
	elements = fields.ElementMap{
		key1: value1,
		key2: value2,
		key3: value3,
		key4: value4,
	}
	properJSON    = []byte(fmt.Sprintf(`{"%s":"%s","%s":%f}`, key1, value1, key2, value2))
	improperJSONs = [][]byte{
		[]byte(fmt.Sprintf(`"%s":"%s",%f:%s}`, key1, value1, value2, key2)),
		[]byte(fmt.Sprintf(`{"%s","%s"}`, key1, key2)),
		[]byte(fmt.Sprintf(`["%s","%s"]`, key1, key2)),
	}
)

func TestAsLabels(t *testing.T) {
	sm := elements.AsLabels()
	assert.Equal(
		t,
		map[string]string{
			"a": "b",
			"c": "6",
			"d": `map[k1:v1 k2:v2]`,
			"e": `[e1 e2]`,
		},
		sm,
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
	for _, key := range []string{key1, key2, key3, key4, key5, key6} {
		if _, exists := elements[key]; exists {
			assert.NotEmpty(t, elements.GetElementAsString(key))
		} else {
			assert.Empty(t, elements.GetElementAsString(key))
		}
	}
}

func TestElementsFromProperJSON(t *testing.T) {
	e, err := fields.ElementMapFromJSON(properJSON)
	assert.NoError(t, err)
	assert.NotNil(t, e)
	assert.Contains(t, e, key1)
	assert.Equal(t, e[key1], value1)
	assert.Contains(t, e, key2)
	assert.Equal(t, e[key2], value2)
}

func TestElementsFromImproperJSON(t *testing.T) {
	for _, JSON := range improperJSONs {
		e, err := fields.ElementMapFromJSON(JSON)
		assert.Error(t, err)
		assert.Nil(t, e)
	}
}

func TestElementsAsString(t *testing.T) {
	expected := `{"a":"b","c":6,"d":{"k1":"v1","k2":"v2"},"e":["e1","e2"]}`
	require.NotNil(t, elements)
	j, marshalErr := json.Marshal(elements)
	assert.NoError(t, marshalErr)
	assert.Equal(t, expected, string(j))
}
