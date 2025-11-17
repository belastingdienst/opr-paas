package groups

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var exampleGroups = `CN=foo,OU=unit1,OU=unit2,DC=belastingdienst,DC=nl
CN=bar,OU=unit1,DC=belastingdienst,DC=nl
CN=baz,OU=unit2,DC=belastingdienst,DC=nl
CN=qux,OU=unit2,DC=belastingdienst,DC=nl`

// Keys() should return a sorted list of group keys.
func TestKeys(t *testing.T) {
	gs := NewGroups()
	gs.AddFromString(exampleGroups)
	k := gs.Keys()

	require.Len(t, k, 4)
	assert.Equal(t, "bar", k[0])
	assert.Equal(t, "baz", k[1])
	assert.Equal(t, "foo", k[2])
	assert.Equal(t, "qux", k[3])
}

// Add() should merge the passed group into the receiver.
func TestAdd(t *testing.T) {
	gs := NewGroups()
	gs.AddFromString(exampleGroups)
	gs2 := NewGroups()
	gs2.AddFromString("CN=additional,OU=org_unit,DC=example,DC=org")
	changed := gs.Add(gs2)
	k := gs.Keys()

	assert.True(t, changed)
	require.Len(t, k, 5)
	assert.Equal(t, "additional", k[0])
}

// Add() should not add duplicate groups.
func TestAddDuplicate(t *testing.T) {
	gs := NewGroups()
	gs.AddFromString(exampleGroups)
	gs2 := NewGroups()
	gs2.AddFromString(exampleGroups)
	changed := gs.Add(gs2)
	k := gs.Keys()

	assert.False(t, changed)
	assert.Len(t, k, 4)
}

// Add() should replace groups with changed queries.
func TestAddReplace(t *testing.T) {
	gs := NewGroups()
	gs.AddFromString(exampleGroups)
	gs2 := NewGroups()
	gs2.AddFromString("CN=foo,OU=unit17,DC=belastingdienst,DC=nl")
	changed := gs.Add(gs2)

	assert.True(t, changed)
	assert.Len(t, gs.Keys(), 4)
	assert.Contains(t, gs.Queries(), "CN=foo,OU=unit17,DC=belastingdienst,DC=nl")
}

// Queries() should return the sorted list of queries.
func TestQueries(t *testing.T) {
	gs := NewGroups()
	gs.AddFromString(exampleGroups)
	q := gs.Queries()

	require.Len(t, q, 4)
	assert.Equal(t, "CN=bar,OU=unit1,DC=belastingdienst,DC=nl", q[0])
	assert.Equal(t, "CN=baz,OU=unit2,DC=belastingdienst,DC=nl", q[1])
}

// DeleteByKey() should delete the passed group.
func TestDeleteByKey(t *testing.T) {
	gs := NewGroups()
	gs.AddFromString(exampleGroups)
	changed := gs.DeleteByKey("foo")
	k := gs.Keys()

	assert.True(t, changed)
	require.Len(t, k, 3)
	assert.NotContains(t, k, "foo")
}

// DeleteByKey() should return false if the key is not present.
func TestDeleteByKeyUnchanged(t *testing.T) {
	gs := NewGroups()
	gs.AddFromString(exampleGroups)
	changed := gs.DeleteByKey("nonexistent")

	assert.False(t, changed)
	assert.Len(t, gs.Keys(), 4)
}

// DeleteByQuery() should delete the group matching the query.
func TestDeleteByQuery(t *testing.T) {
	gs := NewGroups()
	gs.AddFromString(exampleGroups)
	changed := gs.DeleteByQuery("CN=foo,OU=unit1,OU=unit2,DC=belastingdienst,DC=nl")
	k := gs.Keys()

	assert.True(t, changed)
	require.Len(t, k, 3)
	assert.NotContains(t, k, "foo")
}

// DeleteByQuery() should return false if the query is not present.
func TestDeleteByQueryUnchanged(t *testing.T) {
	gs := NewGroups()
	gs.AddFromString(exampleGroups)
	changed := gs.DeleteByQuery("nonexistent")

	assert.False(t, changed)
	assert.Len(t, gs.Keys(), 4)
}
