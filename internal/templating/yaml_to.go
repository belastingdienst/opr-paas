package templating

import (
	"fmt"

	"github.com/belastingdienst/opr-paas/v3/internal/argocd-plugin-generator/fields"
	"gopkg.in/yaml.v3"
)

type (
	// TemplateMapResult is returned by functions parsing maps from yaml, and can be converted to TemplateResult
	TemplateMapResult fields.Elements
	// TemplateListResult is returned by functions parsing lists from yaml, and can be converted to TemplateResult
	TemplateListResult []interface{}
	// TemplateResult is returned when parsing from yaml, it is a map of string keys and values
	TemplateResult map[string]string
)

// AsResult can be used to convert a TemplateMapResult into a TemplateResult (with option to prefix)
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

// AsResult can be used to convert a TemplateListResult into a TemplateResult (with option to prefix)
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

// Merge can be used to merge two TemplateResult's and return te merged TemplateResult
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

// AsFieldElements can be used to convert a TemplateResult into a fields.Elements
func (tmr TemplateResult) AsFieldElements() (result fields.Elements) {
	result = make(fields.Elements)
	for key, value := range tmr {
		result[key] = value
	}
	return result
}

func yamlToMap(data []byte) (result TemplateMapResult, err error) {
	result = make(TemplateMapResult)
	if err = yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func yamlToList(data []byte) (result TemplateListResult, err error) {
	if err = yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
