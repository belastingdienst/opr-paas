package fields

import (
	"fmt"
	"sort"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// Entries represents all entries in the list of the listgenerator
// This is a map so that values are unique, the key is the paas entry
type Entries map[string]Elements

// Merge merges all key/value pairs from another Entries on top of this and returns the resulting total Entries set
func (en Entries) Merge(added Entries) (entries Entries) {
	entries = make(Entries)
	for key, value := range en {
		entries[key] = value
	}
	for key, value := range added {
		if sourceValue, exists := entries[key]; exists {
			entries[key] = sourceValue.Merge(value)
		} else {
			entries[key] = value
		}
	}
	return entries
}

func (en Entries) String() string {
	var l []string
	keys := make([]string, 0, len(en))
	for k := range en {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := en[key]
		l = append(l, fmt.Sprintf("'%s': %s", key, value.String()))
	}
	return fmt.Sprintf("{ %s }", strings.Join(l, ", "))
}

func (en Entries) AsJSON() ([]apiextensionsv1.JSON, error) {
	list := []apiextensionsv1.JSON{}
	keys := make([]string, 0, len(en))
	for k := range en {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		entry := en[key]
		if data, err := entry.AsJSON(); err != nil {
			return nil, err
		} else {
			list = append(list, apiextensionsv1.JSON{Raw: data})
		}
	}
	return list, nil
}

func EntriesFromJSON(data []apiextensionsv1.JSON) (Entries, error) {
	e := Entries{}
	for _, raw := range data {
		if entry, err := ElementsFromJSON(raw.Raw); err != nil {
			return nil, err
		} else {
			key := entry.Key()
			if key != "" {
				e[key] = entry
			} else {
				return nil, fmt.Errorf(`json data "%s" does not contain a "paas" field`, raw)
			}
		}
	}
	return e, nil
}
