package templating

import (
	"bytes"
	"text/template"

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

func (t Templater) Verify(name string, templatedText string) error {
	_, err := template.New(name).Parse(templatedText)
	return err
}

func (t Templater) TemplateToString(name string, templatedText string) (string, error) {
	buf := new(bytes.Buffer)
	tmpl, err := template.New(name).Parse(templatedText)
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
