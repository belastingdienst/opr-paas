package v1alpha2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/belastingdienst/opr-paas/v4/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v4/pkg/fields"
)

var _ = Describe("PaasTypes", func() {
	var paas *v1alpha2.Paas
	const paasName = "mypaas"
	BeforeEach(func() {
		paas = &v1alpha2.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: paasName,
			},
		}
	})
	Describe("New Paas", func() {
		Context("with default values", func() {
			It("should have name properly set", func() {
				Expect(paas.Name).To(Equal(paasName))
			})
		})
	})
})

var _ = Describe("PaasCapability", func() {
	Describe("CapExtraFields", func() {
		It("should return custom fields with defaults applied", func() {
			pc := v1alpha2.PaasCapability{
				CustomFields: map[string]string{
					"git_url":      "https://github.com/org/repo",
					"git_revision": "develop",
				},
			}
			elements := pc.CapExtraFields(map[string]v1alpha2.ConfigCustomField{
				"git_url":      {Validation: "^https://.*$"},
				"git_revision": {},
				"git_path":     {},
				"default_key":  {Default: "default_value"},
			})
			Expect(elements).To(Equal(fields.ElementMap{
				"git_url":      "https://github.com/org/repo",
				"git_revision": "develop",
				"git_path":     "",
				"default_key":  "default_value",
			}))
		})

		It("should silently ignore keys not in config", func() {
			pc := v1alpha2.PaasCapability{
				CustomFields: map[string]string{
					"not_in_config": "breaks",
				},
			}
			elements := pc.CapExtraFields(map[string]v1alpha2.ConfigCustomField{})
			Expect(elements).NotTo(BeNil())
			Expect(elements).NotTo(HaveKey("not_in_config"))
		})

		It("should not validate required fields", func() {
			pc := v1alpha2.PaasCapability{
				CustomFields: map[string]string{},
			}
			elements := pc.CapExtraFields(map[string]v1alpha2.ConfigCustomField{
				"required_key": {Required: true},
			})
			Expect(elements).NotTo(BeNil())
			Expect(elements["required_key"]).To(Equal(""))
		})

		It("should not validate regex patterns", func() {
			pc := v1alpha2.PaasCapability{
				CustomFields: map[string]string{
					"invalid_key": "invalid_value",
				},
			}
			elements := pc.CapExtraFields(map[string]v1alpha2.ConfigCustomField{
				"invalid_key": {Validation: "^valid_value$"},
			})
			Expect(elements).NotTo(BeNil())
			Expect(elements["invalid_key"]).To(Equal("invalid_value"))
		})
	})

	Describe("ValidateCapExtraFields", func() {
		It("should pass validation with valid fields", func() {
			pc := v1alpha2.PaasCapability{
				CustomFields: map[string]string{
					"git_url":      "https://github.com/org/repo",
					"git_revision": "develop",
				},
			}
			err := pc.ValidateCapExtraFields(map[string]v1alpha2.ConfigCustomField{
				"git_url":      {Validation: "^https://.*$"},
				"git_revision": {},
				"default_key":  {Default: "default_value"},
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should error when key is not in config", func() {
			pc := v1alpha2.PaasCapability{
				CustomFields: map[string]string{
					"not_in_config": "breaks",
				},
			}
			err := pc.ValidateCapExtraFields(map[string]v1alpha2.ConfigCustomField{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("custom field not_in_config is not configured in capability config"))
		})

		It("should error when required field is not set", func() {
			pc := v1alpha2.PaasCapability{
				CustomFields: map[string]string{},
			}
			err := pc.ValidateCapExtraFields(map[string]v1alpha2.ConfigCustomField{
				"required_key": {Required: true},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("value required_key is required"))
		})

		It("should error when value does not match validation regex", func() {
			pc := v1alpha2.PaasCapability{
				CustomFields: map[string]string{
					"invalid_key": "invalid_value",
				},
			}
			err := pc.ValidateCapExtraFields(map[string]v1alpha2.ConfigCustomField{
				"invalid_key": {Validation: "^valid_value$"},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid value invalid_value (does not match ^valid_value$)"))
		})
	})
})
