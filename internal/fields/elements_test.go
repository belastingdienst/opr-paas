package fields_test

import (
	"fmt"
	"testing"

	"github.com/belastingdienst/opr-paas/internal/fields"
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
	properJSON    = []byte(fmt.Sprintf(`{"%s":"%s","%s":%f}`, key1, value1, key2, value2))
	improperJSONs = [][]byte{
		[]byte(fmt.Sprintf(`"%s":"%s",%f:%s}`, key1, value1, value2, key2)),
		[]byte(fmt.Sprintf(`{"%s","%s"}`, key1, key2)),
		[]byte(fmt.Sprintf(`["%s","%s"]`, key1, key2)),
	}
)

func TestElementsFromProperJSON(t *testing.T) {
	e, err := fields.ElementsFromJSON(properJSON)
	assert.NoError(t, err)
	assert.NotNil(t, e)
	assert.Contains(t, e, key1)
	assert.Equal(t, e[key1], value1)
	assert.Contains(t, e, key2)
	assert.Equal(t, e[key2], value2)
}

func TestElementsFromImproperJSON(t *testing.T) {
	for _, JSON := range improperJSONs {
		e, err := fields.ElementsFromJSON(JSON)
		assert.Error(t, err)
		assert.Nil(t, e)
	}
}

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

func TestElementsAsString(t *testing.T) {
	expected := `{ 'a': 'b', 'c': '6' }`
	require.NotNil(t, elements)
	assert.Equal(t, expected, elements.String())
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

func TestKey(t *testing.T) {
	const paasName = "my-paas"
	assert.Empty(t, elements.Key())
	elements2 := fields.Elements{
		key1:   value1,
		key2:   value2,
		"paas": paasName,
	}
	assert.Equal(t, paasName, elements2.Key())
}
