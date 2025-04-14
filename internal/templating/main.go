package templating

import (
	"bytes"
	"text/template"

	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/group/all"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
)

// Templater is a struct that can hold a Paas and a PaasConfig and can run go-templates using these as input
type Templater struct {
	Paas   api.Paas
	Config api.PaasConfig
}

// NewTemplater returns an initialized Templater from a Paas and PaasConfig
func NewTemplater(paas api.Paas, config api.PaasConfig) Templater {
	return Templater{
		Paas:   paas,
		Config: config,
	}
}

func (t Templater) getSproutFuncs() (template.FuncMap, error) {
	handler := sprout.New()
	err := handler.AddGroups(all.RegistryGroup())
	if err != nil {
		return nil, err
	}
	return handler.Build(), nil
}

// Verify can verify a template (just parsing it, not running it against a Paas / PaasConfig)
func (t Templater) Verify(name string, templatedText string) error {
	funcs, err := t.getSproutFuncs()
	if err != nil {
		return err
	}
	_, err = template.New(name).Funcs(funcs).Parse(templatedText)
	return err
}

// TemplateToString can be used to parse a go-template and return a string value
func (t Templater) TemplateToString(name string, templatedText string) (string, error) {
	buf := new(bytes.Buffer)
	funcs, err := t.getSproutFuncs()
	if err != nil {
		return "", err
	}
	tmpl, err := template.New(name).Funcs(funcs).Parse(templatedText)
	if err != nil {
		return "", err
	}
	err = tmpl.Execute(buf, t)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// TemplateToMap can be used to parse a go-template and try to parse as map or list.
// If it can be parsed, it will prefix map keys / list indexes by `name` and return the map.
// If it cannot be parsed as map / list, it will return a map with one key, value pair, where key = `name` and value
// is the result.
func (t Templater) TemplateToMap(name string, templatedText string) (result TemplateResult, err error) {
	yamlData, err := t.TemplateToString(name, templatedText)
	if err != nil {
		return nil, err
	}
	myMap, err := yamlToMap([]byte(yamlData))
	if err == nil {
		return myMap.AsResult(name), nil
	}
	myList, err := yamlToList([]byte(yamlData))
	if err == nil {
		return myList.AsResult(name), nil
	}
	return TemplateResult{name: yamlData}, nil
}

// CapCustomFieldsToMap can be used parse all Custom Fields of a Capability and return result in a map of strings
func (t Templater) CapCustomFieldsToMap(capName string) (result TemplateResult, err error) {
	result = make(TemplateResult)
	capConfig := t.Config.Spec.Capabilities[capName]
	for name, fieldConfig := range capConfig.CustomFields {
		if fieldConfig.Template != "" {
			fieldResult, err := t.TemplateToMap(name, fieldConfig.Template)
			if err != nil {
				return nil, err
			}
			result = result.Merge(fieldResult)
		}
	}
	return result, nil
}
