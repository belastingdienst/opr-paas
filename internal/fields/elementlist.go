package fields

import (
	"fmt"
)

// ElementList represents a list of any values
type ElementList []Element

// AsElementMap will convert this any map into a string map
func (el ElementList) AsElementMap() (ElementMap, error) {
	result := ElementMap{}
	for index, value := range el {
		result[fmt.Sprintf("%d", index)] = value
	}
	return result, nil
}
