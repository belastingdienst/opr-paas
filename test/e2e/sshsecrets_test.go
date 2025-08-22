package e2e

import (
	"context"
	"fmt"
	"testing"

	"github.com/belastingdienst/opr-paas-crypttool/pkg/crypt"
	"github.com/belastingdienst/opr-paas/v3/pkg/quota"
	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	paasName        = "sshpaas"
	paasCap1        = "sso"
	paasCap1Ns      = paasName + "-" + paasCap1
	paasCap2        = "tekton"
	argoLabelKey    = "argocd.argoproj.io/secret-type"
	argoLabelValue  = "repo-creds"
	secretTypeKey   = "type"
	secretTypeValue = "git"
	unencrypted     = "updated"
)

func AddCapSSHSecret(t *testing.T, spec *api.PaasSpec, capabilityName string, key string, value string) {
	capability, exists := spec.Capabilities[capabilityName]
	if !exists {
		t.Fatalf("%s should exist but is not defined", paasCap1)
	}
	if capability.Secrets == nil {
		capability.Secrets = map[string]string{key: value}
	} else {
		capability.Secrets[key] = value
	}
	spec.Capabilities[capabilityName] = capability
}

func ResetCapSSHSecret(t *testing.T, spec *api.PaasSpec, capabilityName string) {
	capability, exists := spec.Capabilities[capabilityName]
	if !exists {
		t.Fatalf("%s should exist but is not defined", paasCap1)
	}
	capability.Secrets = nil
	spec.Capabilities[capabilityName] = capability
}

func TestSecrets(t *testing.T) {
	privateKeys, err := crypt.NewPrivateKeysFromFiles([]string{})
	if err != nil {
		panic(fmt.Errorf("unable to create an empty list of private keys: %w", err))
	}
	c, err := crypt.NewCryptFromKeys(privateKeys, "./fixtures/crypt/pub/publicKey0", paasName)
	if err != nil {
		panic(fmt.Errorf("unable to create a crypt: %w", err))
	}

	encrypted, err := c.Encrypt([]byte("rolled"))
	require.NoError(t, err)

	toBeDecryptedPaas := api.PaasSpec{
		Requestor: "paas-user",
		Quota:     make(quota.Quota),
		Secrets:   map[string]string{"ssh://git@scm/some-repo.git": encrypted},
		Capabilities: api.PaasCapabilities{
			paasCap1: api.PaasCapability{
				Secrets: map[string]string{"ssh://git@scm/some-other-repo.git": encrypted},
			},
			paasCap2: api.PaasCapability{Secrets: map[string]string{}},
		},
	}

	testenv.Test(
		t,
		features.New("secrets").
			Setup(createPaasFn(paasName, toBeDecryptedPaas)).
			Assess("is created", assertSecretCreated).
			Assess("is updated when value is updated", assertSecretValueUpdated).
			Assess("is updated when key is updated", assertSecretKeyUpdated).
			Assess("are removed", assertSecretRemovedAfterRemovingFromPaas).
			Teardown(teardownPaasFn(paasName)).
			Feature(),
	)
}

func assertSecretCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasName, t, cfg)
	assert.NotNil(t, paas)
	require.NoError(
		t,
		waitForCondition(ctx, cfg, paas, 0, api.TypeReadyPaas),
		"Paas reconciliation succeeds",
	)

	// Assert secrets
	secret1 := getOrFail(ctx, "paas-ssh-1deb30f1", paasCap1Ns, &corev1.Secret{}, t, cfg)
	secret2 := getOrFail(ctx, "paas-ssh-5c51424e", paasCap1Ns, &corev1.Secret{}, t, cfg)

	assert.NotEmpty(t, secret1)
	assert.NotEmpty(t, secret2)
	// The owner of the Secret is the Paas that created it
	assert.Equal(t, paas.UID, secret1.OwnerReferences[0].UID)
	assert.Equal(t, argoLabelValue, secret1.Labels[argoLabelKey])
	assert.Equal(t, secretTypeValue, string(secret1.Data[secretTypeKey]))
	assert.Equal(t, "ssh://git@scm/some-repo.git", string(secret1.Data["url"]))
	assert.Equal(t, "rolled", string(secret1.Data["sshPrivateKey"]))

	// The owner of the Secret is the Paas that created it
	assert.Equal(t, paas.UID, secret2.OwnerReferences[0].UID)
	assert.Equal(t, argoLabelValue, secret2.Labels[argoLabelKey])
	assert.Equal(t, secretTypeValue, string(secret2.Data[secretTypeKey]))
	assert.Equal(t, "ssh://git@scm/some-other-repo.git", string(secret2.Data["url"]))
	assert.Equal(t, "rolled", string(secret2.Data["sshPrivateKey"]))
	return ctx
}

func assertSecretValueUpdated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	privateKeys, err := crypt.NewPrivateKeysFromFiles([]string{})
	if err != nil {
		panic(fmt.Errorf("unable to create an empty list of private keys: %w", err))
	}
	c, err := crypt.NewCryptFromKeys(privateKeys, "./fixtures/crypt/pub/publicKey0", paasName)
	if err != nil {
		panic(fmt.Errorf("unable to create a crypt: %w", err))
	}

	encrypted, err := c.Encrypt([]byte(unencrypted))
	require.NoError(t, err)

	paas := getPaas(ctx, paasName, t, cfg)
	paas.Spec.Secrets = map[string]string{"ssh://git@scm/some-repo.git": encrypted}
	ResetCapSSHSecret(t, &paas.Spec, paasCap1)
	AddCapSSHSecret(t, &paas.Spec, paasCap1, "ssh://git@scm/some-other-repo.git", encrypted)

	if err = updateSync(ctx, cfg, paas, api.TypeReadyPaas); err != nil {
		t.Fatal(err)
	}

	// List secrets in namespace to be sure
	secrets := &corev1.SecretList{}
	err = cfg.Client().
		Resources().
		List(ctx, secrets, func(opts *k8sv1.ListOptions) { opts.FieldSelector = "metadata.namespace=sshpaas-sso" })
	require.NoError(t, err)
	assert.Len(t, secrets.Items, 2)

	// Assert each secret
	secret1 := getOrFail(ctx, "paas-ssh-1deb30f1", paasCap1Ns, &corev1.Secret{}, t, cfg)
	secret2 := getOrFail(ctx, "paas-ssh-5c51424e", paasCap1Ns, &corev1.Secret{}, t, cfg)

	assert.NotEmpty(t, secret1)
	assert.NotEmpty(t, secret2)
	// The owner of the Secret is the Paas that created it
	assert.Equal(t, paas.UID, secret1.OwnerReferences[0].UID)
	assert.Equal(t, argoLabelValue, secret1.Labels[argoLabelKey])
	assert.Equal(t, secretTypeValue, string(secret1.Data[secretTypeKey]))
	assert.Equal(t, "ssh://git@scm/some-repo.git", string(secret1.Data["url"]))
	assert.Equal(t, unencrypted, string(secret1.Data["sshPrivateKey"]))

	// The owner of the Secret is the Paas that created it
	assert.Equal(t, paas.UID, secret2.OwnerReferences[0].UID)
	assert.Equal(t, argoLabelValue, secret2.Labels[argoLabelKey])
	assert.Equal(t, secretTypeValue, string(secret2.Data[secretTypeKey]))
	assert.Equal(t, "ssh://git@scm/some-other-repo.git", string(secret2.Data["url"]))
	assert.Equal(t, unencrypted, string(secret2.Data["sshPrivateKey"]))

	return ctx
}

func assertSecretKeyUpdated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	privateKeys, err := crypt.NewPrivateKeysFromFiles([]string{})
	if err != nil {
		panic(fmt.Errorf("unable to create an empty list of private keys: %w", err))
	}
	c, err := crypt.NewCryptFromKeys(privateKeys, "./fixtures/crypt/pub/publicKey0", paasName)
	if err != nil {
		panic(fmt.Errorf("unable to create a crypt: %w", err))
	}

	encrypted, err := c.Encrypt([]byte(unencrypted))
	require.NoError(t, err)

	paas := getPaas(ctx, paasName, t, cfg)
	paas.Spec.Secrets = map[string]string{"ssh://git@scm/some-second-repo.git": encrypted}
	ResetCapSSHSecret(t, &paas.Spec, paasCap1)
	AddCapSSHSecret(t, &paas.Spec, paasCap1, "ssh://git@scm/some-other-second-repo.git", encrypted)

	if err = updateSync(ctx, cfg, paas, api.TypeReadyPaas); err != nil {
		t.Fatal(err)
	}

	// List secrets in namespace to be sure
	secrets := &corev1.SecretList{}
	err = cfg.Client().
		Resources().
		List(ctx, secrets, func(opts *k8sv1.ListOptions) { opts.FieldSelector = "metadata.namespace=sshpaas-sso" })
	require.NoError(t, err)
	assert.Len(t, secrets.Items, 2)

	// Assert each secret
	secret1 := getOrFail(ctx, "paas-ssh-6df19938", paasCap1Ns, &corev1.Secret{}, t, cfg)
	secret2 := getOrFail(ctx, "paas-ssh-c1e4bede", paasCap1Ns, &corev1.Secret{}, t, cfg)

	assert.NotEmpty(t, secret1)
	assert.NotEmpty(t, secret2)

	// The owner of the Secret is the Paas that created it
	assert.Equal(t, paas.UID, secret1.OwnerReferences[0].UID)
	assert.Equal(t, argoLabelValue, secret1.Labels[argoLabelKey])
	assert.Equal(t, secretTypeValue, string(secret1.Data[secretTypeKey]))
	assert.Equal(t, "ssh://git@scm/some-second-repo.git", string(secret1.Data["url"]))
	assert.Equal(t, unencrypted, string(secret1.Data["sshPrivateKey"]))

	// The owner of the Secret is the Paas that created it
	assert.Equal(t, paas.UID, secret2.OwnerReferences[0].UID)
	assert.Equal(t, argoLabelValue, secret2.Labels[argoLabelKey])
	assert.Equal(t, secretTypeValue, string(secret2.Data[secretTypeKey]))
	assert.Equal(t, "ssh://git@scm/some-other-second-repo.git", string(secret2.Data["url"]))
	assert.Equal(t, unencrypted, string(secret2.Data["sshPrivateKey"]))

	return ctx
}

func assertSecretRemovedAfterRemovingFromPaas(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasName, t, cfg)
	paas.Spec.Secrets = nil
	cap1, exists := paas.Spec.Capabilities[paasCap1]
	if !exists {
		t.Fatalf("%s should exist but is not defined", paasCap1)
	}
	cap1.Secrets = nil
	paas.Spec.Capabilities[paasCap1] = cap1

	if err := updateSync(ctx, cfg, paas, api.TypeReadyPaas); err != nil {
		t.Fatal(err)
	}

	secrets := &corev1.SecretList{}
	err := cfg.Client().
		Resources().
		List(ctx, secrets, func(opts *k8sv1.ListOptions) { opts.FieldSelector = "metadata.namespace=sshpaas-sso" })
	require.NoError(t, err)
	assert.Empty(t, secrets.Items)

	return ctx
}
