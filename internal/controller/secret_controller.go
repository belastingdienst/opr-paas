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

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
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
func (r *PaasNSReconciler) ensureSecret(
	ctx context.Context,
	secret *corev1.Secret,
) error {
	namespacedName := types.NamespacedName{
		Name:      secret.Name,
		Namespace: secret.Namespace,
	}
	found := &corev1.Secret{}
	err := r.Get(ctx, namespacedName, found)
	if err != nil && errors.IsNotFound(err) {
		return r.Create(ctx, secret)
	} else if err != nil {
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
func (r *PaasNSReconciler) backendSecret(
	ctx context.Context,
	paas *v1alpha1.Paas,
	paasns *v1alpha1.PaasNS,
	namespacedName types.NamespacedName,
	url string,
) (*corev1.Secret, error) {
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
	s.Labels[ManagedByLabelKey] = paas.Name

	logger.Info().Msg("setting Owner")
	err := controllerutil.SetControllerReference(paas, s, r.Scheme)
	if err != nil {
		return s, err
	}
	return s, nil
}

// getSecrets returns a list of Secrets based on the Paas(Ns) spec.
func (r *PaasNSReconciler) getSecrets(
	ctx context.Context,
	paas *v1alpha1.Paas,
	paasns *v1alpha1.PaasNS,
	encryptedSecrets map[string]string,
) (*corev1.SecretList, error) {
	if len(encryptedSecrets) == 0 {
		return &corev1.SecretList{}, nil
	}

	rsa, err := r.getRsa(ctx, paas.Name)
	if err != nil {
		return nil, err
	}

	secrets := &corev1.SecretList{}
	for url, encryptedSecretData := range encryptedSecrets {
		namespacedName := types.NamespacedName{
			Namespace: paasns.NamespaceName(),
			Name:      fmt.Sprintf("paas-ssh-%s", strings.ToLower(hashData(url)[:8])),
		}
		secret, err := r.backendSecret(ctx, paas, paasns, namespacedName, url)
		if err != nil {
			return nil, err
		}
		decrypted, err := rsa.Decrypt(encryptedSecretData)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt secret %s: %s", secret.Name, err.Error())
		}
		secret.Data["sshPrivateKey"] = decrypted
		secrets.Items = append(secrets.Items, *secret)
	}
	return secrets, nil
}

// backendSecrets aggregates SSH secrets from PaasNS and Paas resources.
func (r *PaasNSReconciler) backendSecrets(
	ctx context.Context,
	paasns *v1alpha1.PaasNS,
	paas *v1alpha1.Paas,
) (*corev1.SecretList, error) {
	secrets := make(map[string]string)

	// From the PaasNS resource
	maps.Copy(secrets, paasns.Spec.SSHSecrets)

	// From the Paas Resource capability spec (if applicable)
	if capability, exists := paas.Spec.Capabilities[paasns.Name]; exists && capability.IsEnabled() {
		maps.Copy(secrets, capability.GetSSHSecrets())
	}

	// From the Paas resource
	maps.Copy(secrets, paas.Spec.SSHSecrets)

	return r.getSecrets(ctx, paas, paasns, secrets)
}

// deleteObsoleteSecrets removes secrets that are no longer desired.
func (r *PaasNSReconciler) deleteObsoleteSecrets(
	ctx context.Context,
	existingSecrets *corev1.SecretList,
	desiredSecrets *corev1.SecretList,
) error {
	logger := log.Ctx(ctx)
	logger.Info().Msg("deleting obsolete secrets")

	for _, existingSecret := range existingSecrets.Items {
		if !isSecretInDesiredSecrets(existingSecret, desiredSecrets) {
			if err := r.Delete(ctx, &existingSecret); err != nil {
				logger.Err(err).Str("Secret", existingSecret.Name).Msg("failed to delete Secret")
				return err
			}
			logger.Info().Str("Secret", existingSecret.Name).Msg("deleted obsolete Secret")
		}
	}
	return nil
}

// isSecretInDesiredSecrets checks if a secret exists in the desired list.
func isSecretInDesiredSecrets(secret corev1.Secret, desiredSecrets *corev1.SecretList) bool {
	for _, desiredSecret := range desiredSecrets.Items {
		if secret.Name == desiredSecret.Name && secret.Namespace == desiredSecret.Namespace {
			return true
		}
	}
	return false
}

// getExistingSecrets lists all secrets managed by this Paas in the namespace.
func (r *PaasNSReconciler) getExistingSecrets(
	ctx context.Context,
	paas *v1alpha1.Paas,
	paasns *v1alpha1.PaasNS,
) (*corev1.SecretList, error) {
	logger := log.Ctx(ctx)
	ns := paasns.NamespaceName()
	logger.Debug().Msgf("listing existing secrets in namespace: %s", ns)

	secrets := &corev1.SecretList{}
	opts := []client.ListOption{
		client.InNamespace(ns),
		client.MatchingLabels{ManagedByLabelKey: paas.Name},
	}
	err := r.List(ctx, secrets, opts...)
	if err != nil {
		logger.Err(err).Msg("error listing existing secrets")
		return nil, err
	}
	return secrets, nil
}

// reconcileSecrets ensures that all desired secrets exist and obsolete ones are removed.
func (r *PaasNSReconciler) reconcileSecrets(
	ctx context.Context,
	paas *v1alpha1.Paas,
	paasns *v1alpha1.PaasNS,
) error {
	ctx, logger := logging.GetLogComponent(ctx, "secret")
	logger.Debug().Msg("reconciling SSH secrets")

	desiredSecrets, err := r.backendSecrets(ctx, paasns, paas)
	if err != nil {
		return err
	}
	if desiredSecrets == nil {
		desiredSecrets = &corev1.SecretList{}
	}
	logger.Debug().Int("count", len(desiredSecrets.Items)).Msg("desired secrets count")

	existingSecrets, err := r.getExistingSecrets(ctx, paas, paasns)
	if err != nil {
		return err
	}
	if existingSecrets == nil {
		existingSecrets = &corev1.SecretList{}
	}
	logger.Debug().Int("count", len(existingSecrets.Items)).Msg("existing secrets count")

	if err := r.deleteObsoleteSecrets(ctx, existingSecrets, desiredSecrets); err != nil {
		return err
	}

	for _, secret := range desiredSecrets.Items {
		if err := r.ensureSecret(ctx, &secret); err != nil {
			logger.Err(err).Str("secret", secret.Name).Msg("failure while reconciling secret")
			return err
		}
		logger.Info().Str("secret", secret.Name).Msg("ssh secret successfully reconciled")
	}
	return nil
}
