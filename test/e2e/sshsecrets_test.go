package e2e

import (
	"context"
	"fmt"
	"testing"

	"github.com/belastingdienst/opr-paas/internal/crypt"
	"github.com/belastingdienst/opr-paas/internal/quota"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestSecrets(t *testing.T) {
	privateKeys, err := crypt.NewPrivateKeysFromFiles([]string{})
	if err != nil {
		panic(fmt.Errorf("unable to create an empty list of private keys: %w", err))
	}
	c, err := crypt.NewCryptFromKeys(privateKeys, "./fixtures/crypt/pub/publicKey0", "sshpaas")
	if err != nil {
		panic(fmt.Errorf("unable to create a crypt: %w", err))
	}

	encrypted, err := c.Encrypt([]byte("rolled"))
	require.NoError(t, err)

	toBeDecryptedPaas := api.PaasSpec{
		Requestor:  "paas-user",
		Quota:      make(quota.Quota),
		SshSecrets: map[string]string{"ssh://git@scm/some-repo.git": encrypted},
		Capabilities: api.PaasCapabilities{
			"sso": api.PaasCapability{
				Enabled:    true,
				SshSecrets: map[string]string{"ssh://git@scm/some-other-repo.git": encrypted},
			},
			"tekton": api.PaasCapability{Enabled: true, SshSecrets: map[string]string{}},
		},
	}

	testenv.Test(
		t,
		features.New("secrets").
			Setup(createPaasFn("sshpaas", toBeDecryptedPaas)).
			Assess("is created", assertSecretCreated).
			Assess("is updated when value is updated", assertSecretValueUpdated).
			Assess("is updated when key is updated", assertSecretKeyUpdated).
			Assess("are removed", assertSecretRemovedAfterRemovingFromPaas).
			Teardown(teardownPaasFn("sshpaas")).
			Feature(),
	)
}

func assertSecretCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, "sshpaas", t, cfg)
	assert.NotNil(t, paas)
	// Wait for namespace created by waiting for reconciliation of sso paasns
	ssopaasns := &api.PaasNS{ObjectMeta: v1.ObjectMeta{
		Name:      "sso",
		Namespace: "sshpaas",
	}}
	require.NoError(
		t,
		waitForCondition(ctx, cfg, ssopaasns, 0, api.TypeReadyPaasNs),
		"SSO PaasNS reconciliation succeeds",
	)

	// Assert secrets
	secret1 := getOrFail(ctx, "paas-ssh-1deb30f1", "sshpaas-sso", &corev1.Secret{}, t, cfg)
	secret2 := getOrFail(ctx, "paas-ssh-5c51424e", "sshpaas-sso", &corev1.Secret{}, t, cfg)

	assert.NotEmpty(t, secret1)
	assert.NotEmpty(t, secret2)
	// The owner of the Secret is the Paas that created it
	assert.Equal(t, paas.UID, secret1.OwnerReferences[0].UID)
	assert.Equal(t, "repo-creds", secret1.Labels["argocd.argoproj.io/secret-type"])
	assert.Equal(t, "git", string(secret1.Data["type"]))
	assert.Equal(t, "ssh://git@scm/some-repo.git", string(secret1.Data["url"]))
	assert.Equal(t, "rolled", string(secret1.Data["sshPrivateKey"]))

	// The owner of the Secret is the Paas that created it
	assert.Equal(t, paas.UID, secret2.OwnerReferences[0].UID)
	assert.Equal(t, "repo-creds", secret2.Labels["argocd.argoproj.io/secret-type"])
	assert.Equal(t, "git", string(secret2.Data["type"]))
	assert.Equal(t, "ssh://git@scm/some-other-repo.git", string(secret2.Data["url"]))
	assert.Equal(t, "rolled", string(secret2.Data["sshPrivateKey"]))
	return ctx
}

func assertSecretValueUpdated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	privateKeys, err := crypt.NewPrivateKeysFromFiles([]string{})
	if err != nil {
		panic(fmt.Errorf("unable to create an empty list of private keys: %w", err))
	}
	c, err := crypt.NewCryptFromKeys(privateKeys, "./fixtures/crypt/pub/publicKey0", "sshpaas")
	if err != nil {
		panic(fmt.Errorf("unable to create a crypt: %w", err))
	}

	encrypted, err := c.Encrypt([]byte("updatet"))
	require.NoError(t, err)

	paas := getPaas(ctx, "sshpaas", t, cfg)
	paas.Spec.SshSecrets = map[string]string{"ssh://git@scm/some-repo.git": encrypted}
	if err = paas.Spec.Capabilities.ResetCapSshSecret("sso"); err != nil {
		t.Fatal(err)
	} else if err = paas.Spec.Capabilities.AddCapSshSecret(
		"sso",
		"ssh://git@scm/some-other-repo.git",
		encrypted,
	); err != nil {
		t.Fatal(err)
	}

	oldSsoPaasNs := getOrFail(ctx, "sso", "sshpaas", &api.PaasNS{}, t, cfg)

	if err := updateSync(ctx, cfg, paas, api.TypeReadyPaas); err != nil {
		t.Fatal(err)
	}

	// Wait for reconciliation of sso paasns
	ssopaasns := getOrFail(ctx, "sso", "sshpaas", &api.PaasNS{}, t, cfg)
	require.NoError(
		t,
		waitForCondition(ctx, cfg, ssopaasns, oldSsoPaasNs.Generation, api.TypeReadyPaasNs),
		"SSO PaasNS reconciliation succeeds",
	)

	// List secrets in namespace to be sure
	secrets := &corev1.SecretList{}
	err = cfg.Client().
		Resources().
		List(ctx, secrets, func(opts *v1.ListOptions) { opts.FieldSelector = "metadata.namespace=sshpaas-sso" })
	require.NoError(t, err)
	assert.Len(t, secrets.Items, 2)

	// Assert each secret
	secret1 := getOrFail(ctx, "paas-ssh-1deb30f1", "sshpaas-sso", &corev1.Secret{}, t, cfg)
	secret2 := getOrFail(ctx, "paas-ssh-5c51424e", "sshpaas-sso", &corev1.Secret{}, t, cfg)

	assert.NotEmpty(t, secret1)
	assert.NotEmpty(t, secret2)
	// The owner of the Secret is the Paas that created it
	assert.Equal(t, paas.UID, secret1.OwnerReferences[0].UID)
	assert.Equal(t, "repo-creds", secret1.Labels["argocd.argoproj.io/secret-type"])
	assert.Equal(t, "git", string(secret1.Data["type"]))
	assert.Equal(t, "ssh://git@scm/some-repo.git", string(secret1.Data["url"]))
	assert.Equal(t, "updatet", string(secret1.Data["sshPrivateKey"]))

	// The owner of the Secret is the Paas that created it
	assert.Equal(t, paas.UID, secret2.OwnerReferences[0].UID)
	assert.Equal(t, "repo-creds", secret2.Labels["argocd.argoproj.io/secret-type"])
	assert.Equal(t, "git", string(secret2.Data["type"]))
	assert.Equal(t, "ssh://git@scm/some-other-repo.git", string(secret2.Data["url"]))
	assert.Equal(t, "updatet", string(secret2.Data["sshPrivateKey"]))

	return ctx
}

func assertSecretKeyUpdated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	privateKeys, err := crypt.NewPrivateKeysFromFiles([]string{})
	if err != nil {
		panic(fmt.Errorf("unable to create an empty list of private keys: %w", err))
	}
	c, err := crypt.NewCryptFromKeys(privateKeys, "./fixtures/crypt/pub/publicKey0", "sshpaas")
	if err != nil {
		panic(fmt.Errorf("unable to create a crypt: %w", err))
	}

	encrypted, err := c.Encrypt([]byte("updatet"))
	require.NoError(t, err)

	paas := getPaas(ctx, "sshpaas", t, cfg)
	paas.Spec.SshSecrets = map[string]string{"ssh://git@scm/some-second-repo.git": encrypted}
	if err = paas.Spec.Capabilities.ResetCapSshSecret("sso"); err != nil {
		t.Fatal(err)
	} else if err = paas.Spec.Capabilities.AddCapSshSecret(
		"sso",
		"ssh://git@scm/some-other-second-repo.git",
		encrypted,
	); err != nil {
		t.Fatal(err)
	}

	oldSsoPaasNs := getOrFail(ctx, "sso", "sshpaas", &api.PaasNS{}, t, cfg)

	if err := updateSync(ctx, cfg, paas, api.TypeReadyPaas); err != nil {
		t.Fatal(err)
	}

	// Wait for reconciliation of sso paasns
	ssopaasns := getOrFail(ctx, "sso", "sshpaas", &api.PaasNS{}, t, cfg)
	require.NoError(
		t,
		waitForCondition(ctx, cfg, ssopaasns, oldSsoPaasNs.Generation, api.TypeReadyPaasNs),
		"SSO PaasNS reconciliation succeeds",
	)

	// List secrets in namespace to be sure
	secrets := &corev1.SecretList{}
	err = cfg.Client().
		Resources().
		List(ctx, secrets, func(opts *v1.ListOptions) { opts.FieldSelector = "metadata.namespace=sshpaas-sso" })
	require.NoError(t, err)
	assert.Len(t, secrets.Items, 2)

	// Assert each secret
	secret1 := getOrFail(ctx, "paas-ssh-6df19938", "sshpaas-sso", &corev1.Secret{}, t, cfg)
	secret2 := getOrFail(ctx, "paas-ssh-c1e4bede", "sshpaas-sso", &corev1.Secret{}, t, cfg)

	assert.NotEmpty(t, secret1)
	assert.NotEmpty(t, secret2)

	// The owner of the Secret is the Paas that created it
	assert.Equal(t, paas.UID, secret1.OwnerReferences[0].UID)
	assert.Equal(t, "repo-creds", secret1.Labels["argocd.argoproj.io/secret-type"])
	assert.Equal(t, "git", string(secret1.Data["type"]))
	assert.Equal(t, "ssh://git@scm/some-second-repo.git", string(secret1.Data["url"]))
	assert.Equal(t, "updatet", string(secret1.Data["sshPrivateKey"]))

	// The owner of the Secret is the Paas that created it
	assert.Equal(t, paas.UID, secret2.OwnerReferences[0].UID)
	assert.Equal(t, "repo-creds", secret2.Labels["argocd.argoproj.io/secret-type"])
	assert.Equal(t, "git", string(secret2.Data["type"]))
	assert.Equal(t, "ssh://git@scm/some-other-second-repo.git", string(secret2.Data["url"]))
	assert.Equal(t, "updatet", string(secret2.Data["sshPrivateKey"]))

	return ctx
}

func assertSecretRemovedAfterRemovingFromPaas(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, "sshpaas", t, cfg)
	paas.Spec.SshSecrets = nil
	if err := paas.Spec.Capabilities.ResetCapSshSecret("sso"); err != nil {
		t.Fatal(err)
	}

	oldSsoPaasNs := getOrFail(ctx, "sso", "sshpaas", &api.PaasNS{}, t, cfg)

	if err := updateSync(ctx, cfg, paas, api.TypeReadyPaas); err != nil {
		t.Fatal(err)
	}

	// Wait for reconciliation of sso paasns
	ssopaasns := getOrFail(ctx, "sso", "sshpaas", &api.PaasNS{}, t, cfg)
	require.NoError(
		t,
		waitForCondition(ctx, cfg, ssopaasns, oldSsoPaasNs.Generation, api.TypeReadyPaasNs),
		"SSO PaasNS reconciliation succeeds",
	)

	secrets := &corev1.SecretList{}
	err := cfg.Client().
		Resources().
		List(ctx, secrets, func(opts *v1.ListOptions) { opts.FieldSelector = "metadata.namespace=sshpaas-sso" })
	require.NoError(t, err)
	assert.Empty(t, secrets.Items)

	return ctx
}
