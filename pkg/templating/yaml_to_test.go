package templating

import (
	"testing"

	"github.com/belastingdienst/opr-paas/v4/pkg/fields"
	"github.com/stretchr/testify/assert"
)

func TestYamlToMap(t *testing.T) {
	exampleYaml := `
key1: val1
key2: val2
key3: valc
key4: vald
`
	parsed, err := yamlToMap([]byte(exampleYaml))
	assert.NoError(t, err)
	assert.Equal(t, fields.ElementMap{
		"key1": "val1",
		"key2": "val2",
		"key3": "valc",
		"key4": "vald",
	},
		parsed,
	)
}

func TestResultMerge(t *testing.T) {
	var (
		tr1 = fields.ElementMap{
			"key1": "val1",
			"key2": "val2",
		}
		tr2 = fields.ElementMap{
			"key2": "1",
			"key3": "val3",
		}
		expected = fields.ElementMap{
			"key1": "val1",
			"key2": "1",
			"key3": "val3",
		}
	)
	assert.Equal(t, expected, tr1.Merge(tr2))
}

func TestYamlToList(t *testing.T) {
	exampleYaml := `
- vala
- valb
- val3
- val4
`
	parsed, err := yamlToList([]byte(exampleYaml))
	assert.NoError(t, err)
	expected := fields.ElementList{
		"vala",
		"valb",
		"val3",
		"val4",
	}
	assert.Equal(t, expected, parsed)
}
