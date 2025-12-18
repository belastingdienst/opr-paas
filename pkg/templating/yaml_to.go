package templating

import (
	"github.com/belastingdienst/opr-paas/v4/pkg/fields"
	"github.com/goccy/go-yaml"
)

func yamlToMap(data []byte) (result fields.ElementMap, err error) {
	result = fields.ElementMap{}
	if err = yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func yamlToList(data []byte) (result fields.ElementList, err error) {
	if err = yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
