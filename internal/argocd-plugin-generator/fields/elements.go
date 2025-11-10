package fields

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Element represents a value for one entry in the list of the listgenerator
type Element any

// Elements represents all key, value pairs for one entry in the list of the listgenerator
type Elements map[string]Element

// ElementsFromJSON can be used to import key, value pairs from JSON
func ElementsFromJSON(raw []byte) (Elements, error) {
	newElements := make(Elements)
	if err := json.Unmarshal(raw, &newElements); err != nil {
		return nil, err
	}
	return newElements, nil
}

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
	y, err := yaml.Marshal(element)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(y)), nil
}

// Merge merges all key/value pairs from another Entries on top of this and returns the resulting total Entries set
func (es Elements) Merge(added Elements) Elements {
	for key, value := range added {
		es[key] = value
	}
	return es
}

func (es Elements) String() string {
	var l []string
	keys := make([]string, 0, len(es))
	for k := range es {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := es.GetElementAsString(key)
		key = strings.ReplaceAll(key, "'", "\\'")
		value = strings.ReplaceAll(value, "'", "\\'")
		l = append(l, fmt.Sprintf("'%s': '%s'", key, value))
	}
	return fmt.Sprintf("{ %s }", strings.Join(l, ", "))
}

// AsJSON can be used to export all elements as JSON
func (es Elements) AsJSON() ([]byte, error) {
	return json.Marshal(es)
}

// Key returns the name of the Paas (as derived from the element with name "paas").
// Elements have key, value pairs, and the "paas" value usually exists and has the name of the Paas.
func (es Elements) Key() string {
	if key, exists := es["paas"]; exists {
		paasKey, valid := key.(string)
		if valid {
			return paasKey
		}
	}
	return ""
}
