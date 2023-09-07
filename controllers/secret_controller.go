package controllers

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureSecret ensures Secret presence in given namespace.
func (r *PaasReconciler) EnsureSecret(
	ctx context.Context,
	s *corev1.Secret,
) error {
	// See if namespace exists and create if it doesn't
	found := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      s.Name,
		Namespace: s.Namespace,
	}, found)
	if err != nil && errors.IsNotFound(err) {
		// Create the namespace
		return r.Create(ctx, s)
	} else if err != nil {
		// Error that isn't due to the namespace not existing
		return err
	} else {
		return r.Update(ctx, s)
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
	namespace string,
	url string,
	sshData string,
) *corev1.Secret {
	b64_url := b64Encrypt(url)
	name := fmt.Sprintf("paas-ssh-%s", string(b64_url[:8]))
	logger := getLogger(ctx, paas, "Secret", name)
	logger.Info(fmt.Sprintf("Defining %s Secret", name))

	s := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    paas.ClonedLabels(),
		},
		Data: map[string][]byte{
			"type":          b64Encrypt("git"),
			"url":           b64_url,
			"sshPrivateKey": b64Encrypt(sshData),
		},
	}

	logger.Info("Setting Owner")
	controllerutil.SetControllerReference(paas, s, r.Scheme)
	return s
}

func (r *PaasReconciler) BackendSecrets(
	ctx context.Context,
	paas *v1alpha1.Paas,
) (secrets []*corev1.Secret) {
	var logger logr.Logger
	for _, cap := range paas.Spec.Capabilities.AsMap() {
		if cap.IsEnabled() {
			for url, encryptedSshData := range cap.GetSshSecrets() {
				logger = getLogger(ctx, paas, "Secrets", url)
				if decrypted, err := getRsa(paas.Name).Decrypt(encryptedSshData); err != nil {
					logger.Error(err, "decryption failed")
				} else {
					namespace := fmt.Sprintf("%s-%s", paas.ObjectMeta.Name, cap.CapabilityName())
					secrets = append(secrets, r.backendSecret(ctx, paas, namespace, url, string(decrypted)))
				}
			}
			for url, encryptedSshData := range paas.Spec.SshSecrets {
				logger = getLogger(ctx, paas, "Secrets", url)
				if decrypted, err := getRsa(paas.Name).Decrypt(encryptedSshData); err != nil {
					logger.Error(err, "decryption failed")
				} else {
					namespace := fmt.Sprintf("%s-%s", paas.ObjectMeta.Name, cap.CapabilityName())
					secrets = append(secrets, r.backendSecret(ctx, paas, namespace, url, string(decrypted)))
				}
			}
		}
	}
	return secrets
}
