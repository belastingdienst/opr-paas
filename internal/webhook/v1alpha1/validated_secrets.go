package v1alpha1

import (
	"crypto/sha512"
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/crypt"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type validatedHash [64]byte

func hashFromString(s string) (h validatedHash) {
	return sha512.Sum512([]byte(s))
}

// validatedSecrets is a small helper struct to hold hashes from a Paas / oldPaasNS  and compares secrets
// from another PaasNS. If it is in here it is already validated and does not require validation to safe resources.
type validatedSecrets struct {
	v map[validatedHash]bool
}

// appendFromPaas appends validated secrets from a Paas
func (vs *validatedSecrets) appendFromPaas(paas v1alpha1.Paas) {
	if vs.v == nil {
		vs.v = make(map[validatedHash]bool)
	}
	for _, secret := range paas.Spec.SshSecrets {
		hash := hashFromString(secret)
		vs.v[hash] = true
	}
	for _, cap := range paas.Spec.Capabilities {
		for _, secret := range cap.SshSecrets {
			hash := sha512.Sum512([]byte(secret))
			vs.v[hash] = true
		}
	}
}

// appendFromPaasNS appends validated secrets from a PaasNS
func (vs *validatedSecrets) appendFromPaasNS(paasns v1alpha1.PaasNS) {
	if vs.v == nil {
		vs.v = make(map[validatedHash]bool)
	}
	for _, secret := range paasns.Spec.SshSecrets {
		hash := hashFromString(secret)
		vs.v[hash] = true
	}
}

// Is can be used to check if a hash is already validated
func (vs *validatedSecrets) Is(hash validatedHash) bool {
	_, exists := vs.v[hash]
	return exists
}

// compareSecrets can check all secrets from a PaasNS.
// It first checks against the secrets in the validated struct,
// and if not present in there it uses the getRsaFunc to get a crypt and try decrypting the secret
func (vs validatedSecrets) compareSecrets(unvalidated map[string]string, getRsaFunc func() (*crypt.Crypt, error)) (errs field.ErrorList) {
	// Err when an sshSecret can't be decrypted
	for secretName, secret := range unvalidated {
		if vs.Is(hashFromString(secret)) {
			continue
		}
		if crypt, err := getRsaFunc(); err != nil {
			errs = append(errs, &field.Error{
				Type:   field.ErrorTypeInvalid,
				Field:  field.NewPath("spec").Child("sshSecrets").Key(secretName).String(),
				Detail: fmt.Errorf("failed to get crypt: %w", err).Error(),
			})
		} else if _, err := crypt.Decrypt(secret); err != nil {
			errs = append(errs, &field.Error{
				Type:     field.ErrorTypeInvalid,
				Field:    field.NewPath("spec").Child("sshSecrets").Key(secretName).String(),
				BadValue: secret,
				Detail:   fmt.Errorf("failed to decrypt secret: %w", err).Error(),
			})
		}
	}
	return errs
}
