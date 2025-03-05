package fields

import (
	"fmt"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// Entries represents all entries in the list of the listgenerator
// This is a map so that values are unique, the key is the paas entry
type Entries map[string]Elements

func (en Entries) String() string {
	var l []string
	for key, value := range en {
		l = append(l, fmt.Sprintf("'%s': %s", key, value.String()))
	}
	return fmt.Sprintf("{ %s }", strings.Join(l, ", "))
}

func (en Entries) AsJSON() ([]apiextensionsv1.JSON, error) {
	list := []apiextensionsv1.JSON{}
	for _, entry := range en {
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
				// weird, this entry does not have a paas key, let's preserve, but put aside
				e[string(raw.Raw)] = entry
			}
		}
	}
	return e, nil
}
