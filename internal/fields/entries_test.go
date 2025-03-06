package fields_test

import (
	"testing"

	"github.com/belastingdienst/opr-paas/internal/fields"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

var (
	entries1 = fields.Entries{
		"paas1": fields.Elements{
			"key1": "value1",
			"key2": 2.0,
			"paas": "paas1",
		},
	}
	entries2 = fields.Entries{
		"paas1": fields.Elements{
			"key1": "othervalue1",
			"key4": "othervalue4",
			"paas": "paas1",
		},
		"paas2": fields.Elements{
			"key1": "value1",
			"key3": 3,
			"paas": "paas2",
		},
	}
)

// internal/fields/entries.go						54-56 66-68 72-75

func TestHashData(t *testing.T) {
	resultEntries := entries1.Merge(entries2)
	assert.Len(t, resultEntries, 2)

	require.Contains(t, resultEntries, "paas1")
	elements1 := resultEntries["paas1"]
	assert.Len(t, elements1, 4)
	require.Contains(t, elements1, "key1")
	assert.Equal(t, elements1["key1"], "othervalue1")
	require.Contains(t, elements1, "key2")
	assert.Equal(t, elements1["key2"], 2.0)
	require.Contains(t, elements1, "key4")
	assert.Equal(t, elements1["key4"], "othervalue4")
	assert.Contains(t, elements1, "paas")
	assert.Equal(t, elements1["paas"], "paas1")

	require.Contains(t, resultEntries, "paas2")
	elements2 := resultEntries["paas2"]
	assert.Len(t, elements2, 3)
	require.Contains(t, elements1, "key1")
	assert.Equal(t, "value1", elements2["key1"])
	require.Contains(t, elements2, "key3")
	assert.Equal(t, 3, elements2["key3"])
	assert.Contains(t, elements2, "paas")
	assert.Equal(t, "paas2", elements2["paas"])
}

func TestEntriesString(t *testing.T) {
	assert.Equal(
		t,
		// revive:disable-next-line
		"{ 'paas1': { 'key1': 'othervalue1', 'key2': '2', 'key4': 'othervalue4', 'paas': 'paas1' } }",
		entries1.String(),
	)
	assert.Equal(
		t,
		// revive:disable-next-line
		"{ 'paas1': { 'key1': 'othervalue1', 'key4': 'othervalue4', 'paas': 'paas1' }, 'paas2': { 'key1': 'value1', 'key3': '3', 'paas': 'paas2' } }",
		entries2.String(),
	)
}

func TestEntriesAsJSON(t *testing.T) {
	json1, err := entries1.AsJSON()
	assert.NoError(t, err)
	assert.NotNil(t, json1)
	var expected1 = []v1.JSON{
		v1.JSON{Raw: []byte(`{"key1":"othervalue1","key2":2,"key4":"othervalue4","paas":"paas1"}`)},
	}
	assert.Equal(t, expected1, json1)

	json2, err := entries2.AsJSON()
	assert.NoError(t, err)
	assert.NotNil(t, json2)
	expected2 := []v1.JSON{
		v1.JSON{Raw: []byte(`{"key1":"othervalue1","key4":"othervalue4","paas":"paas1"}`)},
		v1.JSON{Raw: []byte(`{"key1":"value1","key3":3,"paas":"paas2"}`)},
	}
	assert.Equal(t, expected2, json2)
	entries3 := fields.Entries{
		"paas3": fields.Elements{
			"key1": "value1",
		},
	}
	entries3["paas3"]["circular"] = &entries3
	json3, err := entries3.AsJSON()
	assert.Error(t, err)
	assert.Nil(t, json3)
}

func TestEntriesFromJSON(t *testing.T) {
	json1, err := entries1.AsJSON()
	require.NoError(t, err)
	require.NotNil(t, json1)

	entries, err := fields.EntriesFromJSON(json1)
	assert.NoError(t, err)
	assert.Equal(t, entries1, entries)

	for _, jData := range []string{
		`{"key1":"othervalue1","key2":2,"key4":"othervalue4"}`,
		`{"key1":"othervalue1","key2"`,
	} {
		t.Logf("testing with JSON data: %s", jData)
		json2 := []v1.JSON{v1.JSON{Raw: []byte(jData)}}
		entries, err = fields.EntriesFromJSON(json2)
		assert.Error(t, err)
		assert.Nil(t, entries)
	}
}
