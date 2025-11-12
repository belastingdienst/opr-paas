package fields

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
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

// GetElementMapAsStringMap converts all values to a json value and returns the set as a string map
func (em ElementMap) GetElementMapAsStringMap() (map[string]string, error) {
	values := map[string]string{}
	for key, value := range em {
		j, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		values[key] = string(j)
	}
	return values, nil
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
	for key, value := range added {
		em[key] = value
	}
	return em
}

func (em ElementMap) String() string {
	var l []string
	keys := make([]string, 0, len(em))
	for k := range em {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := em.GetElementAsString(key)
		key = strings.ReplaceAll(key, "'", "\\'")
		value = strings.ReplaceAll(value, "'", "\\'")
		l = append(l, fmt.Sprintf("'%s': '%s'", key, value))
	}
	return fmt.Sprintf("{ %s }", strings.Join(l, ", "))
}

// AsJSON can be used to export all elements as JSON
func (em ElementMap) AsJSON() ([]byte, error) {
	return json.Marshal(em)
}

// Key returns the name of the Paas (as derived from the element with name "paas").
// ElementMap have key, value pairs, and the "paas" value usually exists and has the name of the Paas.
func (em ElementMap) Key() string {
	if key, exists := em["paas"]; exists {
		paasKey, valid := key.(string)
		if valid {
			return paasKey
		}
	}
	return ""
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
func (em ElementMap) AsElementMap() (ElementMap, error) {
	return em, nil
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
