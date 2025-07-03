package templating_test

import (
	"fmt"
	"testing"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v3/internal/templating"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
	paas = v1alpha2.Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: paasName,
			UID:  "abc", // Needed or owner references fail
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

func TestVerify(t *testing.T) {
	templater := templating.NewTemplater(paas, paasConfig)
	assert.NoError(t, templater.Verify("for1", "{{ range $group := .Paas.Spec.Groups}}{{$group}}{{end}}"))
	assert.Error(t, templater.Verify("for2", "{{ for $group := .Paas.Spec.Groups}}{{$group}}{{end}}"),
		"for does not exist (should be range)")
	assert.Error(t, templater.Verify("for3", "{{ range $group := .Paas.Spec.Groups}}{{$group}}{{end}"),
		"nor properly terminated")
	assert.NoError(t, templater.Verify("string1", `"0,1,2"`))
	assert.NoError(t, templater.Verify("string2", `"0,1,2`))
	assert.NoError(t, templater.Verify("paasname", `{{ .Paas.Name }}`))
	assert.NoError(t, templater.Verify("paasname2", `{{ .NotAPaas.Name}}`),
		"invalid object names are not yet evaluated")
}

func TestValidTemplateToString(t *testing.T) {
	for i, test := range []struct {
		template string
		expected string
	}{
		{template: "{{ .Paas.Name }}", expected: paasName},
		{template: "{{ .Config.Name }}", expected: paasConfigName},
	} {
		tpl := templating.NewTemplater(paas, paasConfig)
		templated, err := tpl.TemplateToString(fmt.Sprintf("%d", i), test.template)
		assert.NoError(t, err)
		assert.Equal(t, test.expected, templated)
	}
}

func TestInValidTemplateToString(t *testing.T) {
	tpl := templating.NewTemplater(paas, paasConfig)
	templated, err := tpl.TemplateToString("invalid", "{{ .NotAPaas.Name }")
	assert.Error(t, err)
	assert.Empty(t, templated)
	templated, err = tpl.TemplateToString("invalid", "{{ .NotAPaas.Name }}")
	assert.Error(t, err)
	assert.Empty(t, templated)
}

func TestValidTemplateToMap(t *testing.T) {
	for _, test := range []struct {
		key      string
		template string
		expected templating.TemplateResult
	}{
		{
			key:      "mystring",
			template: "{{ .Paas.Name }}",
			expected: templating.TemplateResult{"mystring": paasName},
		},
		{
			key:      "mymap",
			template: `{"a":"b","c":"d"}`,
			expected: templating.TemplateResult{
				"mymap_a": "b",
				"mymap_c": "d",
			},
		},
		{
			key:      "mylist",
			template: `["a","b","c","d"]`,
			expected: templating.TemplateResult{
				"mylist_0": "a",
				"mylist_1": "b",
				"mylist_2": "c",
				"mylist_3": "d",
			},
		},
	} {
		tpl := templating.NewTemplater(paas, paasConfig)
		templated, err := tpl.TemplateToMap(test.key, test.template)
		assert.NoError(t, err)
		assert.Equal(t, test.expected, templated)
	}
}

func TestInValidTemplateToMap(t *testing.T) {
	tpl := templating.NewTemplater(paas, paasConfig)
	templated, err := tpl.TemplateToMap("invalid", "{{ .NotAPaas.Name }")
	assert.Error(t, err)
	assert.Nil(t, templated)
	templated, err = tpl.TemplateToMap("invalid", "{{ .NotAPaas.Name }}")
	assert.Error(t, err)
	assert.Nil(t, templated)
}
