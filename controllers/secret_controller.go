package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
func (r *PaasNSReconciler) EnsureSecret(
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

func hashString(original string) []byte {
	sum := sha256.Sum256([]byte(original))
	return []byte(hex.EncodeToString(sum[:]))
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

func (r *PaasNSReconciler) getSecrets(
	ctx context.Context,
	paas *v1alpha1.Paas,
	nsName string,
	encryptedSecrets map[string]string,
) (secrets []*corev1.Secret) {
	for url, encryptedSecretData := range encryptedSecrets {
		namespacedName := types.NamespacedName{
			Namespace: nsName,
			Name:      fmt.Sprintf("paas-ssh-%s", strings.ToLower(string(hashString(url)[:8]))),
		}
		secret := r.backendSecret(ctx, paas, namespacedName, url)
		if decrypted, err := getRsa(paas.Name).Decrypt(encryptedSecretData); err != nil {
			paas.Status.AddMessage(v1alpha1.PaasStatusError, v1alpha1.PaasStatusParse, secret, err.Error())
		} else {
			paas.Status.AddMessage(v1alpha1.PaasStatusInfo, v1alpha1.PaasStatusParse, secret, "Defining generic secret")
			secret.Data["sshPrivateKey"] = decrypted
			secrets = append(secrets, secret)
		}
	}
	return secrets
}

func (r *PaasNSReconciler) BackendSecrets(
	ctx context.Context,
	paasns *v1alpha1.PaasNS,
	paas *v1alpha1.Paas,
) (secrets []*corev1.Secret) {
	nsName := paasns.NamespaceName()
	// From the PaasNS resource
	secrets = append(secrets, r.getSecrets(ctx, paas, nsName, paasns.Spec.SshSecrets)...)
	//From the Paas Resource capability chapter (if applicable)
	if cap, exists := paas.Spec.Capabilities.AsMap()[paasns.Name]; exists && cap.IsEnabled() {
		secrets = append(secrets, r.getSecrets(ctx, paas, nsName, cap.GetSshSecrets())...)
	}
	// From the Paas resource
	secrets = append(secrets, r.getSecrets(ctx, paas, nsName, paas.Spec.SshSecrets)...)
	return secrets
}
