package fields_test

import (
	"fmt"
	"testing"

	"github.com/belastingdienst/opr-paas/internal/fields"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8sv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	paas1Name   = "paas1"
	paas1Value1 = "p1v1"
	paas1Value2 = 2.0
	paas1Value4 = "p1v4"
	paas2Name   = "paas2"
	paas2Value1 = "p2v1"
	paas2Value2 = 3
	paas2Value3 = "p2v3"
)

var (
	entries1 = fields.Entries{
		paas1Name: fields.Elements{
			"key1": "othervalue1",
			"key2": paas1Value2,
			"paas": paas1Name,
		},
	}
	entries2 = fields.Entries{
		paas1Name: fields.Elements{
			"key1": paas1Value1,
			"key4": paas1Value4,
			"paas": paas1Name,
		},
		paas2Name: fields.Elements{
			"key1": paas2Value1,
			"key3": paas2Value3,
			"paas": paas2Name,
		},
	}
)

// internal/fields/entries.go						54-56 66-68 72-75

func TestHashData(t *testing.T) {
	resultEntries := entries1.Merge(entries2)
	assert.Len(t, resultEntries, 2)

	require.Contains(t, resultEntries, paas1Name)
	elements1 := resultEntries[paas1Name]
	assert.Len(t, elements1, 4)
	require.Contains(t, elements1, "key1")
	assert.Equal(t, elements1["key1"], paas1Value1)
	require.Contains(t, elements1, "key2")
	assert.Equal(t, elements1["key2"], paas1Value2)
	require.Contains(t, elements1, "key4")
	assert.Equal(t, elements1["key4"], paas1Value4)
	assert.Contains(t, elements1, "paas")
	assert.Equal(t, elements1["paas"], paas1Name)

	require.Contains(t, resultEntries, paas2Name)
	elements2 := resultEntries[paas2Name]
	assert.Len(t, elements2, 3)
	require.Contains(t, elements1, "key1")
	assert.Equal(t, paas2Value1, elements2["key1"])
	require.Contains(t, elements2, "key3")
	assert.Equal(t, paas2Value3, elements2["key3"])
	assert.Contains(t, elements2, "paas")
	assert.Equal(t, paas2Name, elements2["paas"])
}

func TestEntriesString(t *testing.T) {
	assert.Equal(
		t,
		fmt.Sprintf("{ '%s': { 'key1': '%s', 'key2': '%.0f', 'key4': '%s', 'paas': '%s' } }",
			paas1Name,
			paas1Value1,
			paas1Value2,
			paas1Value4,
			paas1Name,
		),
		entries1.String(),
	)
	assert.Equal(
		t,
		// revive:disable-next-line
		fmt.Sprintf("{ '%s': { 'key1': '%s', 'key4': '%s', 'paas': '%s' }, '%s': { 'key1': '%s', 'key3': '%s', 'paas': '%s' } }",
			paas1Name,
			paas1Value1,
			paas1Value4,
			paas1Name,
			paas2Name,
			paas2Value1,
			paas2Value3,
			paas2Name,
		),
		entries2.String(),
	)
}

func TestEntriesAsJSON(t *testing.T) {
	json1, err := entries1.AsJSON()
	assert.NoError(t, err)
	assert.NotNil(t, json1)
	expected1 := []k8sv1.JSON{
		{Raw: []byte(
			fmt.Sprintf(`{"key1":"%s","key2":%.0f,"key4":"%s","paas":"%s"}`,
				paas1Value1,
				paas1Value2,
				paas1Value4,
				paas1Name),
		)},
	}
	assert.Equal(t, expected1, json1)

	json2, err := entries2.AsJSON()
	assert.NoError(t, err)
	assert.NotNil(t, json2)
	expected2 := []k8sv1.JSON{
		{Raw: []byte(
			fmt.Sprintf(`{"key1":"%s","key4":"%s","paas":"%s"}`, paas1Value1, paas1Value4, paas1Name),
		)},
		{Raw: []byte(
			fmt.Sprintf(`{"key1":"%s","key3":"%s","paas":"%s"}`,
				paas2Value1,
				paas2Value3,
				paas2Name,
			))},
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
	const (
		paas3Name   = "paas3"
		paas3Value1 = "p3v1"
		paas3Value2 = 2.0
		paas3Value4 = "p3v4"
	)
	json1, err := entries1.AsJSON()
	require.NoError(t, err)
	require.NotNil(t, json1)

	entries, err := fields.EntriesFromJSON(json1)
	assert.NoError(t, err)
	assert.Equal(t, entries1, entries)

	for _, jData := range []string{
		fmt.Sprintf(`{"key1":"%s","key2":%.0f,"key4":"%s"}`, paas3Value1, paas3Value2, paas3Value4),
		fmt.Sprintf(`{"key1":"%s","key2"`, paas1Value1),
	} {
		t.Logf("testing with JSON data: %s", jData)
		json2 := []k8sv1.JSON{{Raw: []byte(jData)}}
		entries, err = fields.EntriesFromJSON(json2)
		assert.Error(t, err)
		assert.Nil(t, entries)
	}
}
