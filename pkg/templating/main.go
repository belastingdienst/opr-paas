package templating

import (
	"bytes"
	"text/template"

	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/group/all"

	"github.com/belastingdienst/opr-paas/v4/api"
	"github.com/belastingdienst/opr-paas/v4/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/v4/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v4/pkg/fields"
)

// PaasUnion is an interface representing either a v1alpha1.Paas or a v1alpha2.Paas
type PaasUnion interface {
	v1alpha1.Paas | v1alpha2.Paas
}

// Templater is a struct that can hold a Paas and a PaasConfig and can run go-templates using these as input
type Templater[P PaasUnion, C api.PaasConfig[S], S any] struct {
	Paas   P
	Config C
}

// NewTemplater returns an initialized Templater from a Paas and PaasConfig
func NewTemplater[P PaasUnion, C api.PaasConfig[S], S any](paas P, config C) Templater[P, C, S] {
	return Templater[P, C, S]{
		Paas:   paas,
		Config: config,
	}
}

func (t Templater[P, C, S]) getSproutFuncs() (template.FuncMap, error) {
	handler := sprout.New()
	err := handler.AddGroups(all.RegistryGroup())
	if err != nil {
		return nil, err
	}
	return handler.Build(), nil
}

// Verify can verify a template (just parsing it, not running it against a Paas / PaasConfig)
func (t Templater[P, C, S]) Verify(name string, templatedText string) error {
	funcs, err := t.getSproutFuncs()
	if err != nil {
		return err
	}
	_, err = template.New(name).Funcs(funcs).Parse(templatedText)
	return err
}

// TemplateToString can be used to parse a go-template and return a string value
func (t Templater[P, C, S]) TemplateToString(name string, templatedText string) (string, error) {
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
func (t Templater[P, C, S]) TemplateToMap(name string, templatedText string) (fields.ElementMap, error) {
	yamlData, templateErr := t.TemplateToString(name, templatedText)
	if templateErr != nil {
		return nil, templateErr
	}
	if myMap, err := yamlToMap([]byte(yamlData)); err == nil {
		return myMap.Prefix(name), nil
	}
	if myList, err := yamlToList([]byte(yamlData)); err == nil {
		return myList.AsElementMap().Prefix(name), nil
	}
	return fields.ElementMap{name: yamlData}, nil
}
