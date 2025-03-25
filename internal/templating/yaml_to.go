package templating

import (
	"fmt"

	"github.com/belastingdienst/opr-paas/internal/fields"
	"gopkg.in/yaml.v3"
)

type (
	TemplateMapResult  fields.Elements
	TemplateListResult []interface{}
	TemplateResult     map[string]string
)

func (tmr TemplateMapResult) AsResult(prefix string) (result TemplateResult) {
	result = make(TemplateResult)
	if prefix != "" {
		prefix += "_"
	}
	for key, value := range tmr {
		result[prefix+key] = fmt.Sprintf("%v", value)
	}
	return result
}

func (tlr TemplateListResult) AsResult(prefix string) (result TemplateResult) {
	result = make(TemplateResult)
	if prefix != "" {
		prefix += "_"
	}
	for i, value := range tlr {
		key := fmt.Sprintf("%s%d", prefix, i)
		result[key] = fmt.Sprintf("%v", value)
	}
	return result
}

func (tmr TemplateResult) Merge(other TemplateResult) (result TemplateResult) {
	result = make(TemplateResult)
	for key, value := range tmr {
		result[key] = value
	}
	for key, value := range other {
		result[key] = value
	}
	return result
}

func (tmr TemplateResult) AsFieldElements() (result fields.Elements) {
	result = make(fields.Elements)
	for key, value := range tmr {
		result[key] = value
	}
	return result
}

func yamlToMap(data []byte) (result TemplateMapResult, err error) {
	result = make(TemplateMapResult)
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func yamlToList(data []byte) (result TemplateListResult, err error) {
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
