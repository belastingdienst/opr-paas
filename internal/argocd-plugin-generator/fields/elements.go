/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package fields

import (
	"errors"
	"fmt"
)

// Entries represents all entries in response of the plugin generator
// This is a map so that values are unique, the key is the paas entry
type Entries map[string]Elements

// Element represents a value for one entry in the list of the listgenerator
type Element any

// Elements represents all key, value pairs for one entry in the list of the listgenerator
type Elements map[string]Element

// GetElementsAsStringMap gets a value and returns as string
// This should be a method on Element, but a method cannot exist on interface datatypes
func (es Elements) GetElementsAsStringMap() (values map[string]string) {
	values = make(map[string]string)
	for key := range es {
		values[key] = es.GetElementAsString(key)
	}
	return values
}

// GetElementAsString gets a value and returns as string
// This should be a method on Element, but a method cannot exist on interface datatypes
func (es Elements) GetElementAsString(key string) string {
	value, err := es.TryGetElementAsString(key)
	if err != nil {
		return ""
	}
	return value
}

// TryGetElementAsString gets a value and returns as string
// This should be a method on Element, but a method cannot exist on interface datatypes
func (es Elements) TryGetElementAsString(key string) (string, error) {
	element, exists := es[key]
	if !exists {
		return "", errors.New("element does not exist")
	}
	value, ok := element.(string)
	if ok {
		return value, nil
	}
	return fmt.Sprintf("%v", element), nil
}

// Merge merges all key/value pairs from another Entries on top of this and returns the resulting total Entries set
func (es Elements) Merge(added Elements) Elements {
	for key, value := range added {
		es[key] = value
	}
	return es
}
