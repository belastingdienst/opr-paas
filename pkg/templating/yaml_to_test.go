package templating

import (
	"github.com/belastingdienst/opr-paas/v5/pkg/fields"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Converting yaml", func() {
	When("converting to map", func() {
		It("should work as expected", func() {
			exampleYaml := "key1: val1\nkey2: val2\nkey3: valc\nkey4: vald"
			parsed, err := yamlToMap([]byte(exampleYaml))
			Expect(err).NotTo(HaveOccurred())
			Expect(parsed).To(Equal(
				fields.ElementMap{
					"key1": "val1",
					"key2": "val2",
					"key3": "valc",
					"key4": "vald",
				}))
		})
	})
	When("merging element map", func() {
		It("should work as expected", func() {
			var (
				tr1 = fields.ElementMap{
					"key1": "val1",
					"key2": "val2",
				}
				tr2 = fields.ElementMap{
					"key2": "1",
					"key3": "val3",
				}
			)
			Expect(tr1.Merge(tr2)).To(Equal(
				fields.ElementMap{
					"key1": "val1",
					"key2": "1",
					"key3": "val3",
				},
			))
		})
	})
	When("converting yaml to list", func() {
		It("should work as expected", func() {
			exampleYaml := "- vala\n- valb\n- val3\n- val4"
			parsed, err := yamlToList([]byte(exampleYaml))
			Expect(err).NotTo(HaveOccurred())
			Expect(parsed).To(Equal(fields.ElementList{
				"vala",
				"valb",
				"val3",
				"val4",
			},
			))
		})
	})
})
