package fields

import (
	"fmt"
)

// ElementList represents a list of any values
type ElementList []Element

// AsElementMap converts the list into an ElementMap with string indices as keys.
func (el ElementList) AsElementMap() ElementMap {
	result := ElementMap{}
	for index, value := range el {
		result[fmt.Sprintf("%d", index)] = value
	}
	return result
}
