package v1alpha1

import "regexp"

type PaasConfigTypeValidations map[string]string
type PaasConfigValidations map[string]PaasConfigTypeValidations

func (pctv PaasConfigTypeValidations) getValidationRE(fieldName string) *regexp.Regexp {
	if validation, exists := pctv[fieldName]; !exists {
		return nil
	} else {
		return regexp.MustCompile(validation)
	}
}

func (pcv PaasConfigValidations) GetValidationRE(crd string, fieldName string) *regexp.Regexp {
	if validations, exists := pcv[crd]; !exists {
		return nil
	} else {
		return validations.getValidationRE(fieldName)
	}
}

func (pc PaasConfig) GetValidationRE(crd string, fieldName string) *regexp.Regexp {
	if pc.Spec.Validations == nil {
		return nil
	} else {
		return pc.Spec.Validations.GetValidationRE(crd, fieldName)
	}
}
