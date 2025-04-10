package v1alpha1

import (
	"fmt"
	"testing"

	"github.com/belastingdienst/opr-paas-crypttool/pkg/crypt"
	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	paasSecret1      = "paasSecret1"
	paasSecret2      = "paasSecret2"
	cap1secret1      = "cap1secret1"
	cap1secret2      = "cap1secret2"
	cap2secret1      = "cap2secret1"
	cap2secret2      = "cap2secret2"
	duplicatedSecret = "duplicatedSecret"
)

func TestValidatedSecretsFromPaas(t *testing.T) {
	validated := validatedSecrets{}
	paas := v1alpha1.Paas{
		Spec: v1alpha1.PaasSpec{
			SSHSecrets: map[string]string{
				paasSecret1:      paasSecret1,
				paasSecret2:      paasSecret2,
				duplicatedSecret: paasSecret2,
			},
			Capabilities: v1alpha1.PaasCapabilities{
				"cap1": v1alpha1.PaasCapability{
					SSHSecrets: map[string]string{
						paasSecret1: cap1secret1,
						paasSecret2: cap1secret2,
					},
				},
				"cap2": v1alpha1.PaasCapability{
					SSHSecrets: map[string]string{
						paasSecret1:            cap2secret1,
						paasSecret2:            cap2secret2,
						duplicatedSecret:       paasSecret2,
						"duplicatedPaasSecret": paasSecret2,
						"duplicatedCap1Secret": cap1secret2,
						"duplicatedCap2Secret": cap2secret2,
					},
				},
			},
		},
	}
	validated.appendFromPaas(paas)
	assert.Len(t, validated.v, 6, "duplicated keys should be there only once")
	for _, secret := range []string{paasSecret1, paasSecret2, cap1secret1, cap1secret2, cap2secret1, cap2secret2} {
		assert.True(t, validated.Is(hashFromString(secret)), "secret '%s' should be validated", secret)
	}
	assert.False(t, validated.Is(hashFromString("invalid")), "secret 'invalid' should not be validated")
}

func TestValidatedSecretsFromPaasNS(t *testing.T) {
	validated := validatedSecrets{}
	paasns := v1alpha1.PaasNS{
		Spec: v1alpha1.PaasNSSpec{
			SSHSecrets: map[string]string{
				paasSecret1:        paasSecret1,
				paasSecret2:        paasSecret2,
				"duplicatedSecret": paasSecret2,
			},
		},
	}
	validated.appendFromPaasNS(paasns)
	assert.Len(t, validated.v, 2, "duplicated keys should be there only once")
	for _, secret := range []string{paasSecret1, paasSecret2} {
		assert.True(t, validated.Is(hashFromString(secret)), "secret '%s' should be validated", secret)
	}
	assert.False(t, validated.Is(hashFromString("invalid")), "secret 'invalid' should not be validated")
}

// When the passed getRsaFunc fails, compareSecrets should return an error
func TestValidatedSecretsCompareRsaError(t *testing.T) {
	unvalidated := map[string]string{
		paasSecret1: cap1secret1,
		paasSecret2: cap1secret2,
	}
	rsaFn := func() (*crypt.Crypt, error) {
		return nil, fmt.Errorf("crypt failure")
	}
	errs := validatedSecrets{}.compareSecrets(unvalidated, rsaFn)

	// We expect 2 errors, one for each secret
	assert.Len(t, errs, 2)
	for _, err := range errs {
		assert.Equal(t, field.ErrorTypeInvalid, err.Type)
		assert.Equal(t, "failed to get crypt: crypt failure", err.Detail)
	}
}
