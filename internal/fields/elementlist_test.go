package fields_test

import (
	"testing"

	"github.com/belastingdienst/opr-paas/v3/internal/fields"
	"github.com/stretchr/testify/assert"
)

var (
	element1     = "a"
	element2     = 6
	element3     = map[string]any{element1: element2}
	element4     = []any{element1, element2}
	listElements = fields.ElementList{
		element1,
		element2,
		element3,
		element4,
	}
)

func TestListAsElementMap(t *testing.T) {
	sm, convErr := listElements.AsElementMap()
	assert.NoError(t, convErr)
	assert.Equal(
		t,
		fields.ElementMap{
			"0": element1,
			"1": element2,
			"2": element3,
			"3": element4,
		},
		sm,
	)
}
