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

// AsJSON can be used to convert Entries into JSON data
func (en Entries) AsJSON() ([]apiextensionsv1.JSON, error) {
	var list []apiextensionsv1.JSON
	keys := make([]string, 0, len(en))
	for k := range en {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		entry := en[key]
		data, err := entry.AsJSON()
		if err != nil {
			return nil, err
		}
		list = append(list, apiextensionsv1.JSON{Raw: data})
	}
	return list, nil
}

// EntriesFromJSON can be used to pass a list of json data and parse it to Entries
func EntriesFromJSON(data []apiextensionsv1.JSON) (Entries, error) {
	e := Entries{}
	for _, raw := range data {
		entry, err := ElementsFromJSON(raw.Raw)
		if err != nil {
			return nil, err
		}
		key := entry.Key()
		if key == "" {
			return nil, fmt.Errorf(`json data "%s" does not contain a "paas" field`, raw)
		}

		e[key] = entry
	}
	return e, nil
}
