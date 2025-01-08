/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"maps"
	"strings"

	"github.com/belastingdienst/opr-paas/internal/crypt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"

	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureSecret ensures Secret presence in given secret.
func (r *PaasNSReconciler) EnsureSecret(
	ctx context.Context,
	secret *corev1.Secret,
) error {
	// See if secret exists and create if it doesn't
	namespacedName := types.NamespacedName{
		Name:      secret.Name,
		Namespace: secret.Namespace,
	}
	found := &corev1.Secret{}
	err := r.Get(ctx, namespacedName, found)
	if err != nil && errors.IsNotFound(err) {
		// Create the secret
		if err = r.Create(ctx, secret); err != nil {
			return err
		}
		return nil
	} else if err != nil {
		// Error that isn't due to the secret not existing
		return err
	} else {
		if err = r.Update(ctx, secret); err != nil {
			return err
		}
		return nil
	}
}

func hashData(original string) string {
	sum := sha512.Sum512([]byte(original))
	return hex.EncodeToString(sum[:])
}

/*
kind: Secret
apiVersion: v1
metadata:
  name: argocd-ssh-c3NoOi8v # c3NoOi8v is first 8 characters of base64 encoded url
  namespace: paa-paa-argocd
  labels:
    argocd.argoproj.io/secret-type: repo-creds
data:
  sshPrivateKey: LS0tLS1C== # Double encrypted data
  type: Z2l0 #git
  url: c3NoOi8vZ2l0QGdpdGh1Yi5jb20vYmVsYXN0aW5nZGllbnN0L215cmVwby8= # ssh://git@github.com/belastingdienst/myrepo/
type: Opaque
*/

// backendSecret is a code for Creating Secret
func (r *PaasNSReconciler) backendSecret(
	ctx context.Context,
	paas *v1alpha1.Paas,
	paasns *v1alpha1.PaasNS,
	namespacedName types.NamespacedName,
	url string,
) (
	*corev1.Secret,
	error,
) {
	logger := log.Ctx(ctx)
	logger.Info().Msg("defining Secret")

	s := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
			Labels:    paasns.ClonedLabels(),
		},
		Data: map[string][]byte{
			"type": []byte("git"),
			"url":  []byte(url),
		},
	}

	s.Labels["argocd.argoproj.io/secret-type"] = "repo-creds"

	logger.Info().Msg("setting Owner")

	err := controllerutil.SetControllerReference(paas, s, r.Scheme)
	if err != nil {
		return s, err
	}
	return s, nil
}

// getSecrets returns a list of Secrets which are desired based on the Paas(Ns) spec
func (r *PaasNSReconciler) getSecrets(
	ctx context.Context,
	paas *v1alpha1.Paas,
	paasns *v1alpha1.PaasNS,
	encryptedSecrets map[string]string,
) (secrets []*corev1.Secret, err error) {
	// Only do something when secrets are required
	if len(encryptedSecrets) > 0 {
		return nil, nil
	}
	// TODO(portly-halicore-76) As we didn't remove DecryptKeysPath yet, that still has to be a valid config option.
	// Therefore we must check if one or the other config has been set.
	// TODO(portly-halicore-76) Check in a decent way, whether this config is empty
	if (GetConfig().DecryptKeysSecret != v1alpha1.NamespacedName{}) {
		// Get the configured decryptSecret
		decryptSecret := &corev1.Secret{}
		err = r.Get(ctx, types.NamespacedName{Name: GetConfig().DecryptKeysSecret.Name, Namespace: GetConfig().DecryptKeysSecret.Namespace}, decryptSecret)
		if err != nil {
			return nil, fmt.Errorf("Unable to get decryptSecret from kubernetes, contact system administrator: %w", err)
		}
		// If the generation is changed, the secret has changed, reset Crypts.
		if decryptSecret.Generation != currentDecryptSecretGeneration {
			resetCrypts()
		}
		// TODO get privateKeys from secret
	} else if GetConfig().DecryptKeyPaths != nil {
		// TODO get privateKeys from path
	} else {
		return nil, fmt.Errorf("no decrypt secret has been configured")
	}

	// TODO create rsa from privateKeys, remove secret from here
	var rsa *crypt.Crypt
	rsa, err = getRsa(paas.Name, *decryptSecret)
	if err != nil {
		return nil, err
	}

	for url, encryptedSecretData := range encryptedSecrets {
		namespacedName := types.NamespacedName{
			Namespace: paasns.NamespaceName(),
			Name:      fmt.Sprintf("paas-ssh-%s", strings.ToLower(hashData(url)[:8])),
		}
		secret, err := r.backendSecret(ctx, paas, paasns, namespacedName, url)
		if err != nil {
			return nil, err
		}
		if decrypted, err := rsa.Decrypt(encryptedSecretData); err != nil {
			return nil, fmt.Errorf("failed to decrypt secret %s: %s", secret, err.Error())
		} else {
			secret.Data["sshPrivateKey"] = decrypted
			secrets = append(secrets, secret)
		}
	}
	return secrets, nil
}

// BackendSecrets returns a list of kubernetes Secrets which are desired based on the Paas(Ns) spec.
// It returns an error when the secrets cannot be determined.
func (r *PaasNSReconciler) BackendSecrets(
	ctx context.Context,
	paasns *v1alpha1.PaasNS,
	paas *v1alpha1.Paas,
) ([]*corev1.Secret, error) {
	secrets := make(map[string]string)

	// From the PaasNS resource
	maps.Copy(secrets, paasns.Spec.SshSecrets)

	// From the Paas Resource capability chapter (if applicable)
	if cap, exists := paas.Spec.Capabilities[paasns.Name]; exists && cap.IsEnabled() {
		maps.Copy(secrets, cap.GetSshSecrets())
	}

	// From the Paas resource
	maps.Copy(secrets, paas.Spec.SshSecrets)

	return r.getSecrets(ctx, paas, paasns, secrets)
}

// deleteObsoleteSecrets deletes any secrets from the existingSecrets which is not listed in the desired secrets.
func (r *PaasNSReconciler) deleteObsoleteSecrets(ctx context.Context, existingSecrets []*corev1.Secret, desiredSecrets []*corev1.Secret) error {
	logger := log.Ctx(ctx)
	logger.Info().Msg("deleting obsolete secrets")

	// Delete secrets that are no longer needed
	for _, existingSecret := range existingSecrets {
		if !isSecretInDesiredSecrets(existingSecret, desiredSecrets) {
			// Secret is not in the desired state, delete it
			if err := r.Delete(ctx, existingSecret); err != nil {
				logger.Err(err).Str("Secret", existingSecret.Name).Msg("failed to delete Secret")
				return err
			}
			logger.Info().Str("Secret", existingSecret.Name).Msg("deleted obsolete Secret")
		}
	}

	return nil
}

// isSecretInDesiredSecrets checks if a given secret exists in the desiredSecrets slice by comparing names and namespaces.
func isSecretInDesiredSecrets(secret *corev1.Secret, desiredSecrets []*corev1.Secret) bool {
	for _, desiredSecret := range desiredSecrets {
		if secret.Name == desiredSecret.Name && secret.Namespace == desiredSecret.Namespace {
			return true
		}
	}
	return false
}

// getExistingSecrets retrieves all secrets owned by this Paas in it's enabled namespaces
func (r *PaasNSReconciler) getExistingSecrets(ctx context.Context, paas *v1alpha1.Paas, paasns *v1alpha1.PaasNS) ([]*corev1.Secret, error) {
	var existingSecrets []*corev1.Secret
	logger := log.Ctx(ctx)
	// Check in NamespaceName
	ns := paasns.NamespaceName()
	logger.Debug().Msgf("listing obsolete secret in namespace: %s", ns)
	var secrets corev1.SecretList
	opts := []client.ListOption{
		client.InNamespace(ns),
	}
	err := r.List(ctx, &secrets, opts...)
	if err != nil {
		logger.Err(err).Msg("error listing existing secrets")
		return []*corev1.Secret{}, err
	}
	logger.Debug().
		Str("ns", ns).
		Int("qty", len(secrets.Items)).
		Msgf("qty of existing secrets in ns")
	for _, secret := range secrets.Items {
		if paas.AmIOwner(secret.OwnerReferences) && strings.HasPrefix(secret.Name, "paas-ssh") {
			logger.Debug().Msg("existing paas-ssh secret")
			existingSecrets = append(existingSecrets, &secret)
			continue
		}
		logger.Debug().Msg("no existing paas-ssh secret")
	}
	logger.Info().Int("secrets", len(existingSecrets)).Msg("qty of existing secrets")
	return existingSecrets, nil
}

func (r *PaasNSReconciler) ReconcileSecrets(
	ctx context.Context,
	paas *v1alpha1.Paas,
	paasns *v1alpha1.PaasNS,
) error {
	ctx = setLogComponent(ctx, "secret")
	logger := log.Ctx(ctx)
	logger.Debug().Msg("reconciling Ssh Secrets")
	desiredSecrets, err := r.BackendSecrets(ctx, paasns, paas)
	if err != nil {
		return err
	}
	existingSecrets, err := r.getExistingSecrets(ctx, paas, paasns)
	if err != nil {
		return err
	}
	err = r.deleteObsoleteSecrets(ctx, existingSecrets, desiredSecrets)
	if err != nil {
		return err
	}
	for _, secret := range desiredSecrets {
		if err := r.EnsureSecret(ctx, secret); err != nil {
			logger.Err(err).Str("secret", secret.Name).Msg("failure while reconciling secret")
			return err
		}
		logger.Info().Str("secret", secret.Name).Msg("ssh secret successfully reconciled")
	}
	return nil
}
