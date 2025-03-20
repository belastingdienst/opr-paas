package templating

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestYamlToMap(t *testing.T) {
	var (
		exampleYaml string = `
key1: val1
key2: val2
key3: valc
key4: vald
`
	)
	parsed, err := yamlToMap([]byte(exampleYaml))
	assert.NoError(t, err)
	expected := TemplateResult{
		"map_key1": "val1",
		"map_key2": "val2",
		"map_key3": "valc",
		"map_key4": "vald",
	}
	assert.Equal(t, expected, parsed.AsResult("map"))
	expected = TemplateResult{
		"key1": "val1",
		"key2": "val2",
		"key3": "valc",
		"key4": "vald",
	}
	assert.Equal(t, expected, parsed.AsResult(""))
}

func TestYamlToList(t *testing.T) {
	var (
		exampleYaml string = `
- vala
- valb
- val3
- val4
`
	)
	parsed, err := yamlToList([]byte(exampleYaml))
	assert.NoError(t, err)
	expected := TemplateResult{
		"list_0": "vala",
		"list_1": "valb",
		"list_2": "val3",
		"list_3": "val4",
	}
	assert.Equal(t, expected, parsed.AsResult("list"))
	expected = TemplateResult{
		"0": "vala",
		"1": "valb",
		"2": "val3",
		"3": "val4",
	}
	assert.Equal(t, expected, parsed.AsResult(""))
}
