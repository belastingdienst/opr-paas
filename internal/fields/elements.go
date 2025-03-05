package fields

import (
	"encoding/json"
	"errors"
	"fmt"
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
	return fmt.Sprintf("%v", value), nil
}

func (es Elements) String() string {
	var l []string
	for key := range es {
		value, err := es.TryGetElementAsString(key)
		if err != nil {
			panic("this is impossible, looping through keys in Elements object, and key cannot be fount, weird.")
		}
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
