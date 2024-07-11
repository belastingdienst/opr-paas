/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package groups

import (
	"sort"
	"strings"
)

// Simple struct to parse a string into a map of groups with key is cn so it will be unique
// struct can add and struct can be changed back into a string.
type Groups struct {
	by_key map[string]Group
}

type Group struct {
	Key   string
	Query string
}

func NewGroups() *Groups {
	return &Groups{
		by_key: make(map[string]Group),
	}
}

func (gs *Groups) DeleteByKey(key string) bool {
	if _, exists := gs.by_key[key]; exists {
		delete(gs.by_key, key)
		return true
	}
	return false
}

func (gs *Groups) DeleteByQuery(query string) bool {
	for key, value := range gs.by_key {
		if value.Query == query {
			delete(gs.by_key, key)
			return true
		}
	}
	return false
}

func (gs *Groups) Add(other *Groups) bool {
	var changed bool
	for key, value := range other.by_key {
		if newVal, exists := gs.by_key[key]; !exists {
			changed = true
		} else if newVal != value {
			changed = true
		}
		gs.by_key[key] = value
	}
	return changed
}

func NewGroup(query string) *Group {
	// CN=gkey,OU=org_unit,DC=example,DC=org
	if cn := strings.Split(string(query), ",")[0]; !strings.ContainsAny(cn, "=") {
		return nil
	} else {
		return &Group{
			Key:   strings.SplitN(cn, "=", 2)[1],
			Query: query,
		}
	}
}

func (gs *Groups) AddFromStrings(l []string) {
	for _, query := range l {
		group := NewGroup(query)
		if group != nil {
			gs.by_key[group.Key] = *group
		}
	}
}

func (gs *Groups) AddFromString(s string) {
	gs.AddFromStrings(strings.Split(s, "\n"))
}

func (gs Groups) Keys() []string {
	keys := make([]string, 0, len(gs.by_key))
	for _, group := range gs.by_key {
		keys = append(keys, group.Key)
	}
	sort.Strings(keys)
	return keys
}

func (gs Groups) Queries() []string {
	queries := make([]string, 0, len(gs.by_key))
	for _, group := range gs.by_key {
		queries = append(queries, group.Query)
	}
	sort.Strings(queries)
	return queries
}

func (gs Groups) AsString() string {
	return strings.Join(gs.Queries(), "\n")
}
