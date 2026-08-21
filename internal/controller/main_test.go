/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("testing intersection", func() {
	When("finding the intersection between 2 lists", func() {
		var (
			l1 = []string{"v1", "v2", "v2", "v3", "v4"}
			l2 = []string{"v2", "v2", "v3", "v5"}
		)
		It("should work as expected", func() {
			li := intersect(l1, l2)
			// Expected to have only all values that exist in list 1 and 2, only once (unique)
			Expect(li).To(Equal([]string{"v2", "v3"}))
		})
	})
})
