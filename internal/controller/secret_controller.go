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
	"strings"

	"github.com/belastingdienst/opr-paas/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/internal/logging"

	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureSecret ensures Secret presence in given secret.
func (r *PaasReconciler) ensureSecret(
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
		return r.Create(ctx, secret)
	} else if err != nil {
		// Error that isn't due to the secret not existing
		return err
	}

	return r.Update(ctx, secret)
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
func (r *PaasReconciler) backendSecret(
	ctx context.Context,
	paas *v1alpha2.Paas,
	paasns *v1alpha2.PaasNS,
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
			Labels:    map[string]string{},
		},
		Data: map[string][]byte{
			"type": []byte("git"),
			"url":  []byte(url),
		},
	}
	if paasns != nil {
		s.Labels = paasns.ClonedLabels()
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
func (r *PaasReconciler) backendSecrets(
	ctx context.Context,
	paas *v1alpha2.Paas,
	paasns *v1alpha2.PaasNS,
	namespace string,
	encryptedSecrets map[string]string,
) (secrets []*corev1.Secret, err error) {
	// Only do something when secrets are required
	if len(encryptedSecrets) == 0 {
		return nil, nil
	}

	rsa, err := r.getRsa(ctx, paas.Name)
	if err != nil {
		return nil, err
	}

	for url, encryptedSecretData := range encryptedSecrets {
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      fmt.Sprintf("paas-ssh-%s", strings.ToLower(hashData(url)[:8])),
		}
		secret, err := r.backendSecret(ctx, paas, paasns, namespacedName, url)
		if err != nil {
			return nil, err
		}
		decrypted, err := rsa.Decrypt(encryptedSecretData)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt secret %s: %s", secret, err.Error())
		}
		secret.Data["sshPrivateKey"] = decrypted
		secrets = append(secrets, secret)
	}
	return secrets, nil
}

// deleteObsoleteSecrets deletes any secrets from the existingSecrets which is not listed in the desired secrets.
func (r *PaasReconciler) deleteObsoleteSecrets(
	ctx context.Context,
	existingSecrets []*corev1.Secret,
	desiredSecrets []*corev1.Secret,
) error {
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

// isSecretInDesiredSecrets checks if a given secret exists in the desiredSecrets slice
// by comparing names and namespaces.
func isSecretInDesiredSecrets(secret *corev1.Secret, desiredSecrets []*corev1.Secret) bool {
	for _, desiredSecret := range desiredSecrets {
		if secret.Name == desiredSecret.Name && secret.Namespace == desiredSecret.Namespace {
			return true
		}
	}
	return false
}

// getExistingSecrets retrieves all secrets owned by this Paas in it's enabled namespaces
func (r *PaasReconciler) getExistingSecrets(
	ctx context.Context,
	paas *v1alpha2.Paas,
	ns string,
) ([]*corev1.Secret, error) {
	var existingSecrets []*corev1.Secret
	logger := log.Ctx(ctx)
	// Check in NamespaceName
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

func (r *PaasReconciler) reconcileNamespaceSecrets(
	ctx context.Context,
	paas *v1alpha2.Paas,
	paasns *v1alpha2.PaasNS,
	namespace string,
	paasSecrets map[string]string,
) error {
	ctx, logger := logging.GetLogComponent(ctx, "secret")
	logger.Debug().Msg("reconciling Ssh Secrets")
	desiredSecrets, err := r.backendSecrets(ctx, paas, paasns, namespace, paasSecrets)
	if err != nil {
		return err
	}
	logger.Debug().Int("count", len(desiredSecrets)).Msg("desired secrets count")
	existingSecrets, err := r.getExistingSecrets(ctx, paas, namespace)
	if err != nil {
		return err
	}
	logger.Debug().Int("count", len(existingSecrets)).Msg("existing secrets count")
	err = r.deleteObsoleteSecrets(ctx, existingSecrets, desiredSecrets)
	if err != nil {
		return err
	}
	for _, secret := range desiredSecrets {
		if err := r.ensureSecret(ctx, secret); err != nil {
			logger.Err(err).Str("secret", secret.Name).Msg("failure while reconciling secret")
			return err
		}
		logger.Info().Str("secret", secret.Name).Msg("ssh secret successfully reconciled")
	}
	return nil
}

func (r *PaasReconciler) reconcilePaasSecrets(
	ctx context.Context,
	paas *v1alpha2.Paas,
	nsDefs namespaceDefs,
) error {
	for _, nsDef := range nsDefs {
		err := r.reconcileNamespaceSecrets(ctx, paas, nsDef.paasns, nsDef.nsName, nsDef.secrets)
		if err != nil {
			return err
		}
	}
	return nil
}
