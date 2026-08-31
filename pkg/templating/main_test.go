package templating_test

import (
	"fmt"

	"github.com/belastingdienst/opr-paas/v5/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v5/pkg/fields"
	"github.com/belastingdienst/opr-paas/v5/pkg/templating"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Templating", func() {
	const (
		paasName        = "to-be-templated"
		paasConfigName  = "paas-config"
		capName         = "paasCap"
		group1          = "g1"
		group1Query     = "q1"
		group2          = "g2"
		customField1Key = "cf1"
		customField2Key = "cf2"
	)
	var (
		group2Users = []string{
			"user1",
			"user2",
		}
		group1Roles = []string{
			"role1",
			"role2",
		}
		group2Roles = []string{
			"role3",
			"role4",
		}
		labels = fields.ElementMap{
			"lbl1": "some",
			"lbl2": "thing",
		}
		paas = v1alpha2.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name:   paasName,
				UID:    "abc", // Needed or owner references fail
				Labels: labels.AsLabels(),
			},
			Spec: v1alpha2.PaasSpec{
				Requestor: capName,
				Capabilities: v1alpha2.PaasCapabilities{
					capName: v1alpha2.PaasCapability{},
				},
				Groups: v1alpha2.PaasGroups{
					group1: v1alpha2.PaasGroup{Query: group1Query, Roles: group1Roles},
					group2: v1alpha2.PaasGroup{Users: group2Users, Roles: group2Roles},
				},
			},
		}
		paasConfig = v1alpha2.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: paasConfigName,
			},
			Spec: v1alpha2.PaasConfigSpec{
				Capabilities: map[string]v1alpha2.ConfigCapability{
					capName: {
						CustomFields: map[string]v1alpha2.ConfigCustomField{
							customField1Key: {},
							customField2Key: {},
						},
					},
				},
			},
		}
	)
	When("verifying", func() {
		It("should work as expected", func() {
			templater := templating.NewTemplater(paas, paasConfig)
			Expect(templater.Verify("for1", "{{ range $group := .Paas.Spec.Groups}}{{$group}}{{end}}")).
				NotTo(HaveOccurred())
			Expect(templater.Verify("for2", "{{ for $group := .Paas.Spec.Groups}}{{$group}}{{end}}")).To(
				MatchError(ContainSubstring("function \"for\" not defined")))

			Expect(templater.Verify("for3", "{{ range $group := .Paas.Spec.Groups}}{{$group}}{{end}")).To(
				MatchError(ContainSubstring("bad character")))

			Expect(templater.Verify("string1", `"0,1,2"`)).NotTo(HaveOccurred())
			Expect(templater.Verify("string2", `"0,1,2`)).NotTo(HaveOccurred())
			Expect(templater.Verify("paasname", `{{ .Paas.Name }}`)).NotTo(HaveOccurred())
			Expect(templater.Verify("paasname2", `{{ .NotAPaas.Name }}`)).To(
				MatchError(ContainSubstring("can't evaluate field NotAPaas")))
		})
	})
	When("templating to string", func() {
		It("should work as expected for valid templates", func() {
			tpl := templating.NewTemplater(paas, paasConfig)
			for i, test := range []struct {
				template string
				expected string
			}{
				{template: "{{ .Paas.Name }}", expected: paasName},
				{template: "{{ .Config.Name }}", expected: paasConfigName},
			} {
				templated, err := tpl.TemplateToString(fmt.Sprintf("%d", i), test.template)
				Expect(err).NotTo(HaveOccurred())
				Expect(templated).To(Equal(test.expected))
			}
		})
		It("should return err for invalid templates", func() {
			tpl := templating.NewTemplater(paas, paasConfig)
			templated, err := tpl.TemplateToString("invalid", "{{ .NotAPaas.Name }")
			Expect(err).To(MatchError(ContainSubstring("unexpected \"}\" in operand")))
			Expect(templated).To(BeEmpty())
		})
	})
	When("templating to map", func() {
		It("should work as expected for valid templates", func() {
			for _, test := range []struct {
				key      string
				template string
				expected fields.ElementMap
			}{
				{
					key:      "mystring",
					template: "{{ .Paas.Name }}",
					expected: fields.ElementMap{"mystring": paasName},
				},
				{
					key:      "mymap",
					template: `{"a":"b","c":"d"}`,
					expected: fields.ElementMap{
						"mymap-a": "b",
						"mymap-c": "d",
					},
				},
				{
					key:      "mylist",
					template: `["a","b","c","d"]`,
					expected: fields.ElementMap{
						"mylist-0": "a",
						"mylist-1": "b",
						"mylist-2": "c",
						"mylist-3": "d",
					},
				},
				{
					key:      "object",
					template: "{{ toYAML .Paas.ObjectMeta.Labels }}",
					expected: labels.Prefix("object"),
				},
			} {
				tpl := templating.NewTemplater(paas, paasConfig)
				templated, err := tpl.TemplateToMap(test.key, test.template)
				Expect(err).NotTo(HaveOccurred())
				Expect(templated).To(Equal(test.expected))
			}
		})
		It("should return err for invalid templates", func() {
			tpl := templating.NewTemplater(paas, paasConfig)
			templated, err := tpl.TemplateToMap("invalid", "{{ .NotAPaas.Name }")
			Expect(err).To(MatchError(ContainSubstring("unexpected \"}\" in operand")))
			Expect(templated).To(BeEmpty())
		})
	})
	When("using extrafuncs", func() {
		It("should work as expected", func() {
			var (
				expected   = "extra string"
				extraFunc  = func() string { return expected }
				extraFuncs = map[string]any{"extraFunc": extraFunc}
			)
			tpl := templating.NewTemplater(paas, paasConfig, extraFuncs)
			templated, err := tpl.TemplateToString("extrafunc test", "{{ extraFunc }}")
			Expect(err).NotTo(HaveOccurred())
			Expect(templated).To(Equal(expected))
		})
	})
	When("getting mapstringmapstring", func() {
		It("should work as expected", func() {
			var (
				expected = map[string]map[string]string{
					"s1": {"k1": "v1", "k2": "v2"},
				}
				extraFunc  = func() map[string]map[string]string { return expected }
				extraFuncs = map[string]any{"smsms": extraFunc}
			)
			tpl := templating.NewTemplater(paas, paasConfig, extraFuncs)
			templated, err := tpl.TemplateToStringMapStringMap("stringmapstringmap", "{{ smsms | toYAML }}")
			Expect(err).NotTo(HaveOccurred())
			Expect(templated).To(Equal(expected))
		})
	})
})
