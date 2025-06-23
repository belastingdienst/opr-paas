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

// Groups holds multiple groups in a key value store, where the key is derived from the group name
type Groups struct {
	byKey map[string]Group
}

// Group is a simple struct with a key, and an LDAP query
type Group struct {
	Key   string
	Query string
}

// NewGroups return an empty Groups object
func NewGroups() *Groups {
	return &Groups{
		byKey: make(map[string]Group),
	}
}

// DeleteByKey finds a Group by Key and deletes if it exists
func (gs *Groups) DeleteByKey(key string) bool {
	if _, exists := gs.byKey[key]; exists {
		delete(gs.byKey, key)
		return true
	}
	return false
}

// DeleteByQuery finds a Group by Query and deletes if it exists
func (gs *Groups) DeleteByQuery(query string) bool {
	for key, value := range gs.byKey {
		if value.Query == query {
			delete(gs.byKey, key)
			return true
		}
	}
	return false
}

// Add adds two groups and returns the combined version (and if it was changed)
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

// NewGroup creates a new Group from a Query (deriving the Name from the Query)
func NewGroup(query string) *Group {
	// CN=gkey,OU=org_unit,DC=example,DC=org
	cn := strings.Split(query, ",")[0]
	if !strings.ContainsAny(cn, "=") {
		return nil
	}
	return &Group{
		Key:   strings.SplitN(cn, "=", 2)[1],
		Query: query,
	}
}

// AddFromStrings creates new groups from a list of Queries and adds them
func (gs *Groups) AddFromStrings(l []string) {
	for _, query := range l {
		group := NewGroup(query)
		if group != nil {
			gs.byKey[group.Key] = *group
		}
	}
}

// AddFromString creates a new group from a Query (string) and adds it
func (gs *Groups) AddFromString(s string) {
	gs.AddFromStrings(strings.Split(s, "\n"))
}

// Keys returns a list of all keys
func (gs Groups) Keys() []string {
	keys := make([]string, 0, len(gs.byKey))
	for _, group := range gs.byKey {
		keys = append(keys, group.Key)
	}
	sort.Strings(keys)
	return keys
}

// Queries returns a list of all Queries
func (gs Groups) Queries() []string {
	queries := make([]string, 0, len(gs.byKey))
	for _, group := range gs.byKey {
		queries = append(queries, group.Query)
	}
	sort.Strings(queries)
	return queries
}

// AsString returns a single string with all Queries joined by "\n"
func (gs Groups) AsString() string {
	return strings.Join(gs.Queries(), "\n")
}
