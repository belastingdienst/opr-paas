package fields

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Elements represents all key, value pars for one entry in the list of the listgenerator
type Element any

type Elements map[string]Element

func ElementsFromJSON(raw []byte) (Elements, error) {
	newElements := make(Elements)
	if err := json.Unmarshal(raw, &newElements); err != nil {
		return nil, err
	} else {
		return newElements, nil
	}
}

// GetElementsAsString gets a value and returns as string
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

func (es Elements) AsJSON() ([]byte, error) {
	return json.Marshal(es)
}

func (es Elements) Key() string {
	if key, exists := es["paas"]; exists {
		paasKey, valid := key.(string)
		if valid {
			return paasKey
		}
	}
	return ""
}
