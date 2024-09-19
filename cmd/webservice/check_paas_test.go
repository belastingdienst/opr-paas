package main

import (
	"os"
	"testing"

	v1alpha1 "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCheckPaas(t *testing.T) {
	// generate private/public keys
	priv, err := os.CreateTemp("", "private")
	require.NoError(t, err, "Creating tempfile for private key")
	defer os.Remove(priv.Name()) // clean up

	pub, err := os.CreateTemp("", "public")
	require.NoError(t, err, "Creating tempfile for public key")
	defer os.Remove(pub.Name()) // clean up

	crypt.NewGeneratedCrypt(priv.Name(), pub.Name()) //nolint:errcheck // this is fine in test

	getConfig()
	_config.PublicKeyPath = pub.Name()
	_config.PrivateKeyPath = priv.Name()
	assert.Nil(t, _crypt)
	rsa := getRsa("paasName")

	encrypted, err := rsa.Encrypt([]byte("My test string"))
	require.NoError(t, err)

	toBeDecryptedPaas := &v1alpha1.Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: "paasName",
		},
		Spec: v1alpha1.PaasSpec{
			SshSecrets: map[string]string{"ssh://git@scm/some-repo.git": encrypted},
			Capabilities: v1alpha1.PaasCapabilities{
				SSO: v1alpha1.PaasSSO{Enabled: true, SshSecrets: map[string]string{"ssh://git@scm/some-repo.git": encrypted}},
			},
		},
	}

	err = CheckPaas(rsa, toBeDecryptedPaas)
	require.NoError(t, err)

	notTeBeDecryptedPaas := &v1alpha1.Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: "paasName",
		},
		Spec: v1alpha1.PaasSpec{SshSecrets: map[string]string{"ssh://git@scm/some-repo.git": "bm90RGVjcnlwdGFibGU="}},
	}

	// Must be able to decrypt this
	err = CheckPaas(rsa, notTeBeDecryptedPaas)
	require.Error(t, err)

	partialToBeDecrypedPaas := &v1alpha1.Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: "paasName",
		},
		Spec: v1alpha1.PaasSpec{
			SshSecrets: map[string]string{"ssh://git@scm/some-repo.git": encrypted},
			Capabilities: v1alpha1.PaasCapabilities{
				SSO: v1alpha1.PaasSSO{Enabled: true, SshSecrets: map[string]string{"ssh://git@scm/some-repo.git": "bm90RGVjcnlwdGFibGU="}},
			},
		},
	}

	// Must be able to decrypt this
	err = CheckPaas(rsa, partialToBeDecrypedPaas)
	require.Error(t, err)
}
