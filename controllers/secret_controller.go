package controllers

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureSecret ensures Secret presence in given secret.
func (r *PaasReconciler) EnsureSecret(
	ctx context.Context,
	paas *v1alpha1.Paas,
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
			paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusCreate, secret, err.Error())
		} else {
			paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusCreate, secret, "succeeded")
		}
		return err
	} else if err != nil {
		// Error that isn't due to the secret not existing
		paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusFind, secret, err.Error())
		return err
	} else {
		if err = r.Update(ctx, secret); err != nil {

			paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusUpdate, secret, err.Error())
		} else {
			paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusUpdate, secret, "succeeded")

		}
		return err
	}
}

func b64Encrypt(original string) []byte {
	b64 := make([]byte, base64.StdEncoding.EncodedLen(len(original)))
	base64.StdEncoding.Encode(b64, []byte(original))
	return b64
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
	paas *v1alpha1.Paas,
	namespacedName types.NamespacedName,
	url string,
) *corev1.Secret {
	logger := getLogger(ctx, paas, "Secret", namespacedName.String())
	logger.Info("Defining Secret")

	s := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
			Labels:    paas.ClonedLabels(),
		},
		Data: map[string][]byte{
			"type": []byte("git"),
			"url":  []byte(url),
		},
	}

	s.Labels["argocd.argoproj.io/secret-type"] = "repo-creds"

	logger.Info("Setting Owner")
	controllerutil.SetControllerReference(paas, s, r.Scheme)
	return s
}

func (r *PaasReconciler) BackendSecrets(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (secrets []*corev1.Secret) {
	for _, cap := range paas.Spec.Capabilities.AsMap() {
		logger := getLogger(ctx, paas, "Secrets", cap.CapabilityName())
		if cap.IsEnabled() {
			for url, encryptedSshData := range cap.GetSshSecrets() {
				namespacedName := types.NamespacedName{
					Namespace: fmt.Sprintf("%s-%s", paas.ObjectMeta.Name, cap.CapabilityName()),
					Name:      fmt.Sprintf("paas-ssh-%s", strings.ToLower(string(b64Encrypt(url)[:8]))),
				}
				secret := r.backendSecret(ctx, paas, namespacedName, url)
				if decrypted, err := getRsa(paas.Name).Decrypt(encryptedSshData); err != nil {
					logger.Info(fmt.Sprintf("decryption failed: %e", err))
					paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusParse, secret, err.Error())
				} else {
					logger.Info("Defining secret", "url", url)
					secret.Data["sshPrivateKey"] = decrypted
					secrets = append(secrets, secret)
				}
			}
			for url, encryptedSshData := range paas.Spec.SshSecrets {
				namespacedName := types.NamespacedName{
					Namespace: fmt.Sprintf("%s-%s", paas.ObjectMeta.Name, cap.CapabilityName()),
					Name:      fmt.Sprintf("paas-ssh-%s", strings.ToLower(string(b64Encrypt(url)[:8]))),
				}
				secret := r.backendSecret(ctx, paas, namespacedName, url)
				if decrypted, err := getRsa(paas.Name).Decrypt(encryptedSshData); err != nil {
					logger.Info(fmt.Sprintf("decryption failed: %e", err))
					paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusParse, secret, err.Error())
				} else {
					logger.Info("Defining generic secret", "url", url)
					secret.Data["sshPrivateKey"] = decrypted
					secrets = append(secrets, secret)
				}
			}
		}
	}
	return secrets
}
