/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain_intersection(t *testing.T) {
	l1 := []string{"v1", "v2", "v2", "v3", "v4"}
	l2 := []string{"v2", "v2", "v3", "v5"}
	li := intersect(l1, l2)
	// Expected to have only all values that exist in list 1 and 2, only once (unique)
	lExpected := []string{"v2", "v3"}
	assert.ElementsMatch(t, li, lExpected, "result of intersection not as expected")
}
