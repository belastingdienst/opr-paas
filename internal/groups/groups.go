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
	byKey map[string]Group
}

type Group struct {
	Key   string
	Query string
}

func NewGroups() *Groups {
	return &Groups{
		byKey: make(map[string]Group),
	}
}

func (gs *Groups) DeleteByKey(key string) bool {
	if _, exists := gs.byKey[key]; exists {
		delete(gs.byKey, key)
		return true
	}
	return false
}

func (gs *Groups) DeleteByQuery(query string) bool {
	for key, value := range gs.byKey {
		if value.Query == query {
			delete(gs.byKey, key)
			return true
		}
	}
	return false
}

func (gs *Groups) Add(other *Groups) bool {
	var changed bool
	for key, value := range other.byKey {
		if newVal, exists := gs.byKey[key]; !exists {
			changed = true
		} else if newVal != value {
			changed = true
		}
		gs.byKey[key] = value
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
			gs.byKey[group.Key] = *group
		}
	}
}

func (gs *Groups) AddFromString(s string) {
	gs.AddFromStrings(strings.Split(s, "\n"))
}

func (gs Groups) Keys() []string {
	keys := make([]string, 0, len(gs.byKey))
	for _, group := range gs.byKey {
		keys = append(keys, group.Key)
	}
	sort.Strings(keys)
	return keys
}

func (gs Groups) Queries() []string {
	queries := make([]string, 0, len(gs.byKey))
	for _, group := range gs.byKey {
		queries = append(queries, group.Query)
	}
	sort.Strings(queries)
	return queries
}

func (gs Groups) AsString() string {
	return strings.Join(gs.Queries(), "\n")
}
