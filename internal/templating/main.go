package templating

import (
	"bytes"
	"github.com/Masterminds/sprig/v3"
	"text/template"

	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/group/all"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
)

type Templater struct {
	Paas   api.Paas
	Config api.PaasConfig
}

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

func (t Templater) Verify(name string, templatedText string) error {
	funcs, err := t.getSproutFuncs()
	if err != nil {
		return err
	}
	_, err = template.New(name).Funcs(funcs).Parse(templatedText)
	return err
}

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
