package v1alpha1

import "regexp"

// Configure custom validations for a specific CRD (e.a. paas, paasConfig or PaasNs).
// Refer to https://belastingdienst.github.io/opr-paas/latest/administrators-guide/validations/ for more info.
type PaasConfigTypeValidations map[string]string

// PaasConfigValidations is a map which holds all validations,
// with key being the (lower case) name of the crd and value being a PaasConfigTypeValidations object.
type PaasConfigValidations map[string]PaasConfigTypeValidations

// getValidationRE is an internal function which checks if a validation RE is configured
// and returns a Regexp object if it is, or nil if it isn't
func (pctv PaasConfigTypeValidations) getValidationRE(fieldName string) *regexp.Regexp {
	if validation, exists := pctv[fieldName]; !exists {
		return nil
	} else {
		return regexp.MustCompile(validation)
	}
}

// GetValidationRE can be used to get a validation for a crd by name
// and returns a Regexp object if it is, or nil if it isn't
func (pcv PaasConfigValidations) GetValidationRE(crd string, fieldName string) *regexp.Regexp {
	if validations, exists := pcv[crd]; !exists {
		return nil
	} else {
		return validations.getValidationRE(fieldName)
	}
}

// GetValidationRE can be used to get a validation for a crd by name
// and returns a Regexp object if it is, or nil if it isn't
// This method exists for a PaasConfig and for a PaasConfigValidations, where the former is safe to use
// even when paasConfig.Spec.Validations is not set (making it nil)
func (pc PaasConfig) GetValidationRE(crd string, fieldName string) *regexp.Regexp {
	if pc.Spec.Validations == nil {
		return nil
	} else {
		return pc.Spec.Validations.GetValidationRE(crd, fieldName)
	}
}
