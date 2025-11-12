package fields

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
)

// ElementMap represents all key, value pairs for one entry in the list of the listgenerator
type ElementMap map[string]Element

// ElementMapFromJSON can be used to import key, value pairs from JSON
func ElementMapFromJSON(raw []byte) (ElementMap, error) {
	newElementMap := make(ElementMap)
	if err := json.Unmarshal(raw, &newElementMap); err != nil {
		return nil, err
	}
	return newElementMap, nil
}

// GetElementAsString gets a value and returns as string
// This should be a method on Element, but a method cannot exist on interface datatypes
func (em ElementMap) GetElementAsString(key string) string {
	value, err := em.TryGetElementAsString(key)
	if err != nil {
		return ""
	}
	return value
}

// TryGetElementAsString gets a value and returns as string
// This should be a method on Element, but a method cannot exist on interface datatypes
func (em ElementMap) TryGetElementAsString(key string) (string, error) {
	element, exists := em[key]
	if !exists {
		return "", errors.New("element does not exist")
	}
	value, ok := element.(string)
	if ok {
		return value, nil
	}
	j, err := json.Marshal(element)
	if err != nil {
		return "", err
	}
	return string(j), nil
}

// Merge merges all key/value pairs from another Entries on top of this and returns the resulting total Entries set
func (em ElementMap) Merge(added ElementMap) ElementMap {
	merged := maps.Clone(em)
	for key, value := range added {
		merged[key] = value
	}
	return merged
}

// AsJSON can be used to export all elements as JSON
func (em ElementMap) AsJSON() ([]byte, error) {
	return json.Marshal(em)
}

// AsLabels will convert this any map into a string map
func (em ElementMap) AsLabels() map[string]string {
	result := map[string]string{}
	for key, value := range em {
		result[key] = fmt.Sprintf("%v", value)
	}
	return result
}

// AsElementMap will convert this any map into a string map
func (em ElementMap) AsElementMap() ElementMap {
	return em
}

// Prefix will return a new ElementMap with all keys prefixed with a value
func (em ElementMap) Prefix(prefix string) ElementMap {
	prefixed := ElementMap{}
	for key, value := range em {
		if key == "" {
			key = prefix
		} else if prefix != "" {
			key = fmt.Sprintf("%s-%s", prefix, key)
		}
		prefixed[key] = value
	}
	return prefixed
}
