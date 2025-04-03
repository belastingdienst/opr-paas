package templating_test

import (
	"fmt"
	"testing"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/templating"
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
	paas = api.Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: paasName,
			UID:  "abc", // Needed or owner references fail
		},
		Spec: api.PaasSpec{
			Requestor: capName,
			Capabilities: api.PaasCapabilities{
				capName: api.PaasCapability{
					Enabled: true,
				},
			},
			Groups: api.PaasGroups{
				group1: api.PaasGroup{Query: group1Query, Roles: group1Roles},
				group2: api.PaasGroup{Users: group2Users, Roles: group2Roles},
			},
		},
	}
	paasConfig = api.PaasConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: paasConfigName,
		},
		Spec: api.PaasConfigSpec{
			Capabilities: map[string]api.ConfigCapability{
				capName: {
					CustomFields: map[string]api.ConfigCustomField{
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

func TestCapCustomFieldsToMap(t *testing.T) {
	const (
		myTemplate = `system:cluster-admins, role:admin
{{range $groupName, $groupConfig := .Paas.Spec.Groups }}g, {{ $groupName }}, role:admin
{{end}}`
	)
	var (
		paas = api.Paas{
			ObjectMeta: metav1.ObjectMeta{
				Name: "paas",
			},
			Spec: api.PaasSpec{
				Groups: api.PaasGroups{
					"my-group-1": api.PaasGroup{},
					"my-group-2": api.PaasGroup{},
				},
			},
		}
		paasConfig = api.PaasConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-config",
			},
			Spec: api.PaasConfigSpec{
				Capabilities: api.ConfigCapabilities{
					"my-cap": api.ConfigCapability{
						CustomFields: map[string]api.ConfigCustomField{
							"argocd-policy": {
								Template: myTemplate,
							},
						},
					},
				},
			},
		}
		expected = templating.TemplateResult{
			"argocd-policy": `system:cluster-admins, role:admin
g, my-group-1, role:admin
g, my-group-2, role:admin
`,
		}
	)
	tpl := templating.NewTemplater(paas, paasConfig)
	tplResults, err := tpl.CapCustomFieldsToMap("my-cap")
	assert.NoError(t, err)
	assert.Equal(t, expected, tplResults)
}
