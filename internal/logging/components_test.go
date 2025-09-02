package logging

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComponentToString(t *testing.T) {
	var nonexistentComponent Component = -1
	assert.Equal(t, componentToString(nonexistentComponent), componentToString(UnknownComponent))
}

func TestNewComponentsFromString(t *testing.T) {
	var (
		components = []string{
			"plugin_generator",
			"config_watcher",
			"undefined_component",
			"unittest_component",
		}
		commaSeparatedString = strings.Join(components, ",")
		result               = NewComponentsFromString(commaSeparatedString)
	)
	for _, compName := range components {
		comp, exists := componentConverter[compName]
		require.True(t, exists)
		require.Contains(t, result, comp)
		assert.True(t, result[comp])
	}
}

func TestNewComponentsFromStringMap(t *testing.T) {
	var (
		trueComponents = []string{
			"plugin_generator",
			"undefined_component",
		}
		falseComponents = []string{
			"config_watcher",
			"unittest_component",
		}
		strMap = map[string]bool{}
	)
	for _, compName := range trueComponents {
		strMap[compName] = true
	}
	for _, compName := range falseComponents {
		strMap[compName] = false
	}
	result := NewComponentsFromStringMap(strMap)

	for _, compName := range trueComponents {
		comp, exists := componentConverter[compName]
		require.True(t, exists)
		enabled, exists := result[comp]
		require.True(t, exists)
		require.True(t, enabled)
	}
	for _, compName := range falseComponents {
		comp, exists := componentConverter[compName]
		require.True(t, exists)
		enabled, exists := result[comp]
		require.True(t, exists)
		require.False(t, enabled)
	}
}
