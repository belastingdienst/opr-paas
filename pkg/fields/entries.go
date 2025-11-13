package fields

import (
	"fmt"
	"sort"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// Entries represents all entries in the list of the listgenerator
// This is a map so that values are unique, the key is the paas entry
type Entries map[string]ElementMap

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

// FromJSON can be used to pass a list of json data and fill the values of this Entries with the result
func (en *Entries) FromJSON(key string, data []apiextensionsv1.JSON) error {
	self := *en
	for _, raw := range data {
		entry, err := ElementMapFromJSON(raw.Raw)
		if err != nil {
			return err
		}
		value, exists := entry[key]
		if !exists {
			return fmt.Errorf(`json data "%s" does not contain a "%s" field`, raw, key)
		}
		name, ok := value.(string)
		if !ok {
			return fmt.Errorf(`json data "%s" has a "%s" field, but it is not a string`, raw, key)
		}
		self[name] = entry
	}
	return nil
}
