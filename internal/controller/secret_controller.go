/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"fmt"

	"github.com/belastingdienst/opr-paas/v5/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v5/internal/config"
	"github.com/belastingdienst/opr-paas/v5/internal/logging"
	"github.com/belastingdienst/opr-paas/v5/pkg/templating"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	backwardsCompatibleTemplate = `
      {{- $secrets := dict -}}
      {{- range $key, $value := getPaasSecrets -}}
        {{- $hash := $key | sha512Sum | trunc 8 -}}
        {{- $secretName := print "paas-ssh-" $hash -}}
        {{- $secretData := dict "type" "git" "url" $key "sshPrivateKey" $value -}}
        {{- $_ := $secrets | set $secretName $secretData -}}
      {{- end -}}
      {{- $secrets | toYAML -}}`
)

// ensureSecret ensures Secret presence in given secret.
func (r *PaasReconciler) ensureSecret(
	ctx context.Context,
	secret *corev1.Secret,
) error {
	// See if secret exists and create if it doesn't
	found := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKeyFromObject(secret), found)
	if err != nil && errors.IsNotFound(err) {
		// Create the secret
		return r.Create(ctx, secret)
	} else if err != nil {
		// Error that isn't due to the secret not existing
		return err
	}

	return r.Update(ctx, secret)
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
	data map[string]string,
) (*corev1.Secret, error) {
	_, logger := logging.GetLogComponent(ctx, logging.ControllerSecretComponent)
	logger.Info().Msg("defining Secret")
	bData := map[string][]byte{}
	for k, v := range data {
		bData[k] = []byte(v)
	}
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
			Labels:    map[string]string{},
		},
		Data: bData,
	}
	if paasns != nil {
		s.Labels = paasns.ClonedLabels()
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

// getSecrets returns a list of Secrets which are desired based on the Paas(Ns) spec
func (r *PaasReconciler) backendSecrets(
	ctx context.Context,
	paas *v1alpha2.Paas,
	paasns *v1alpha2.PaasNS,
	namespace string,
	secretsData map[string]map[string]string,
) (*corev1.SecretList, error) {
	var err error

	secrets := &corev1.SecretList{}
	for secretName, secretData := range secretsData {
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      secretName,
		}
		var secret *corev1.Secret
		secret, err = r.backendSecret(ctx, paas, paasns, namespacedName, secretData)
		if err != nil {
			return nil, err
		}
		secrets.Items = append(secrets.Items, *secret)
	}
	return secrets, nil
}

// deleteObsoleteSecrets deletes any secrets from the existingSecrets which is not listed in the desired secrets.
func (r *PaasReconciler) deleteObsoleteSecrets(
	ctx context.Context,
	existingSecrets *corev1.SecretList,
	desiredSecrets *corev1.SecretList,
) error {
	ctx, logger := logging.GetLogComponent(ctx, logging.ControllerSecretComponent)
	logger.Info().Msg("deleting obsolete secrets")

	// Delete secrets that are no longer needed
	for _, existingSecret := range existingSecrets.Items {
		if !isSecretInDesiredSecrets(existingSecret, desiredSecrets) {
			// Secret is not in the desired state, delete it
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

// getExistingSecrets retrieves all secrets managed by this Paas in the namespace.
func (r *PaasReconciler) getExistingSecrets(
	ctx context.Context,
	paas *v1alpha2.Paas,
	ns string,
) (*corev1.SecretList, error) {
	ctx, logger := logging.GetLogComponent(ctx, logging.ControllerSecretComponent)
	// Check in NamespaceName
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

func (r *PaasReconciler) reconcileNamespaceSecrets(
	ctx context.Context,
	paas *v1alpha2.Paas,
	paasns *v1alpha2.PaasNS,
	namespace string,
	encryptedSecrets map[string]map[string]string,
) error {
	ctx, logger := logging.GetLogComponent(ctx, logging.ControllerSecretComponent)
	logger.Debug().Msg("reconciling Secrets")
	desiredSecrets, err := r.backendSecrets(ctx, paas, paasns, namespace, encryptedSecrets)
	if err != nil {
		return err
	}
	if desiredSecrets == nil {
		desiredSecrets = &corev1.SecretList{}
	}
	logger.Debug().Int("count", len(desiredSecrets.Items)).Msg("desired secrets count")

	existingSecrets, err := r.getExistingSecrets(ctx, paas, namespace)
	if err != nil {
		return err
	}
	if existingSecrets != nil {
		logger.Debug().Int("count", len(existingSecrets.Items)).Msg("existing secrets count")
		if err = r.deleteObsoleteSecrets(ctx, existingSecrets, desiredSecrets); err != nil {
			return err
		}
	}

	for _, secret := range desiredSecrets.Items {
		if err = r.ensureSecret(ctx, &secret); err != nil {
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
	// The nsDefs contains the desired namespaces. When obsolete namespaces are deleted, that cascade deletes
	// the secrets in that namespace.
	decryptFunc, err := r.getDecryptFunc(ctx, paas.Name)
	if err != nil {
		return err
	}
	paasConfig, err := config.GetConfigFromContext(ctx)
	if err != nil {
		return err
	}

	for _, nsDef := range nsDefs {
		getPaasSecret := func(key string) (string, error) {
			secret, ok := nsDef.encryptedSecrets[key]
			if !ok {
				return "", fmt.Errorf("secret '%s' does not exist", key)
			}
			decrypted, err := decryptFunc(secret)
			if err != nil {
				return "", err
			}
			return decrypted, nil
		}
		getPaasSecrets := func() (map[string]string, error) {
			secrets := map[string]string{}
			for name, secret := range nsDef.encryptedSecrets {
				decrypted, err := decryptFunc(secret)
				if err != nil {
					return nil, err
				}
				secrets[name] = decrypted
			}
			return secrets, nil
		}
		secretFuncs := map[string]any{
			"decryptPaasSecret": decryptFunc,
			"getPaasSecret":     getPaasSecret,
			"getPaasSecrets":    getPaasSecrets,
		}
		templater := templating.NewTemplater(*paas, paasConfig, secretFuncs)
		var template string
		if nsDef.capName == "" {
			template = paasConfig.Spec.NamespaceSecrets
		} else {
			capConfig, ok := paasConfig.Spec.Capabilities[nsDef.capName]
			if !ok {
				return fmt.Errorf("capability %s is not configured", nsDef.capName)
			}
			template = capConfig.Secrets
		}
		// TODO - Devotional Phoenix - For now we default to old behavior, but on next major we should enforce setting a
		// template rather than defaulting to original behavior which is wrong in most cases
		if template == "" {
			template = backwardsCompatibleTemplate
		}
		secrets, tplErr := templater.TemplateToStringMapStringMap(nsDef.nsName, template)
		if tplErr != nil {
			return tplErr
		}
		reconcileErr := r.reconcileNamespaceSecrets(ctx, paas, nsDef.paasns, nsDef.nsName, secrets)
		if reconcileErr != nil {
			return reconcileErr
		}
	}
	return nil
}
