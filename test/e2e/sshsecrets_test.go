package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

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
	c, err := crypt.NewCrypt([]string{"/tmp/paas-e2e/secrets/priv"}, "/tmp/paas-e2e/secrets/pub/publicKey0", "sshpaas")
	if err != nil {
		panic(fmt.Errorf("unable to create a crypt: %w", err))
	}

	encrypted, err := c.Encrypt([]byte("rolled"))
	require.NoError(t, err)

	toBeDecryptedPaas := api.PaasSpec{
		Requestor:  "paas-user",
		Quota:      make(quota.Quotas),
		SshSecrets: map[string]string{"ssh://git@scm/some-repo.git": encrypted},
		Capabilities: api.PaasCapabilities{
			SSO: api.PaasSSO{Enabled: true, SshSecrets: map[string]string{"ssh://git@scm/some-other-repo.git": encrypted}},
		},
	}

	testenv.Test(
		t,
		features.New("secrets").
			Setup(createPaasFn("sshpaas", toBeDecryptedPaas)).
			Assess("is created", assertSecretCreated).
			Assess("is updated when value is updated", assertSecretValueUpdated).
			Assess("is updated when key is updated", assertSecretKeyUpdated).
			Assess("is not removed", assertSecretNotRemovedAfterRemovingFromPaas).
			Teardown(teardownPaasFn("sshpaas")).
			Feature(),
	)
}

func assertSecretCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, "sshpaas", t, cfg)
	assert.NotNil(t, paas)
	// Wait for namespace created by waiting for reconciliation of sso paasns
	ssopaasns := getOrFail(ctx, "sso", "sshpaas", &api.PaasNS{}, t, cfg)
	require.NoError(t, waitForPaasNSReconciliation(ctx, cfg, ssopaasns), "SSO PaasNS reconciliation succeeds")

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
	c, err := crypt.NewCrypt([]string{"/tmp/paas-e2e/secrets/priv"}, "/tmp/paas-e2e/secrets/pub/publicKey0", "sshpaas")
	if err != nil {
		panic(fmt.Errorf("unable to create a crypt: %w", err))
	}

	encrypted, err := c.Encrypt([]byte("updatet"))
	require.NoError(t, err)

	paas := getPaas(ctx, "sshpaas", t, cfg)
	paas.Spec.SshSecrets = map[string]string{"ssh://git@scm/some-repo.git": encrypted}
	paas.Spec.Capabilities.SSO.SshSecrets = map[string]string{"ssh://git@scm/some-other-repo.git": encrypted}

	if err := updatePaasSync(ctx, cfg, paas); err != nil {
		t.Fatal(err)
	}

	// Wait for reconciliation of sso paasns
	ssopaasns := getOrFail(ctx, "sso", "sshpaas", &api.PaasNS{}, t, cfg)
	// TODO (portly-halicore-76) wait better for this but how?
	time.Sleep(10 * time.Second)
	require.NoError(t, waitForPaasNSReconciliation(ctx, cfg, ssopaasns), "SSO PaasNS reconciliation succeeds")

	// List secrets in namespace to be sure
	secrets := &corev1.SecretList{}
	err = cfg.Client().Resources().List(ctx, secrets, func(opts *v1.ListOptions) { opts.FieldSelector = "metadata.namespace=sshpaas-sso" })
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
	c, err := crypt.NewCrypt([]string{"/tmp/paas-e2e/secrets/priv"}, "/tmp/paas-e2e/secrets/pub/publicKey0", "sshpaas")
	if err != nil {
		panic(fmt.Errorf("unable to create a crypt: %w", err))
	}

	encrypted, err := c.Encrypt([]byte("updatet"))
	require.NoError(t, err)

	paas := getPaas(ctx, "sshpaas", t, cfg)
	paas.Spec.SshSecrets = map[string]string{"ssh://git@scm/some-second-repo.git": encrypted}
	paas.Spec.Capabilities.SSO.SshSecrets = map[string]string{"ssh://git@scm/some-other-second-repo.git": encrypted}

	if err := updatePaasSync(ctx, cfg, paas); err != nil {
		t.Fatal(err)
	}

	// Wait for reconciliation of sso paasns
	ssopaasns := getOrFail(ctx, "sso", "sshpaas", &api.PaasNS{}, t, cfg)
	// TODO (portly-halicore-76) wait better for this but how?
	time.Sleep(10 * time.Second)
	require.NoError(t, waitForPaasNSReconciliation(ctx, cfg, ssopaasns), "SSO PaasNS reconciliation succeeds")

	// List secrets in namespace to be sure
	secrets := &corev1.SecretList{}
	err = cfg.Client().Resources().List(ctx, secrets, func(opts *v1.ListOptions) { opts.FieldSelector = "metadata.namespace=sshpaas-sso" })
	require.NoError(t, err)
	assert.Len(t, secrets.Items, 4)

	// Assert each secret
	secret1 := getOrFail(ctx, "paas-ssh-1deb30f1", "sshpaas-sso", &corev1.Secret{}, t, cfg)
	secret2 := getOrFail(ctx, "paas-ssh-5c51424e", "sshpaas-sso", &corev1.Secret{}, t, cfg)
	secret3 := getOrFail(ctx, "paas-ssh-6df19938", "sshpaas-sso", &corev1.Secret{}, t, cfg)
	secret4 := getOrFail(ctx, "paas-ssh-c1e4bede", "sshpaas-sso", &corev1.Secret{}, t, cfg)

	assert.NotEmpty(t, secret1)
	assert.NotEmpty(t, secret2)
	assert.NotEmpty(t, secret3)
	assert.NotEmpty(t, secret4)

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

	// The owner of the Secret is the Paas that created it
	assert.Equal(t, paas.UID, secret3.OwnerReferences[0].UID)
	assert.Equal(t, "repo-creds", secret3.Labels["argocd.argoproj.io/secret-type"])
	assert.Equal(t, "git", string(secret3.Data["type"]))
	assert.Equal(t, "ssh://git@scm/some-second-repo.git", string(secret3.Data["url"]))
	assert.Equal(t, "updatet", string(secret3.Data["sshPrivateKey"]))

	// The owner of the Secret is the Paas that created it
	assert.Equal(t, paas.UID, secret4.OwnerReferences[0].UID)
	assert.Equal(t, "repo-creds", secret4.Labels["argocd.argoproj.io/secret-type"])
	assert.Equal(t, "git", string(secret4.Data["type"]))
	assert.Equal(t, "ssh://git@scm/some-other-second-repo.git", string(secret4.Data["url"]))
	assert.Equal(t, "updatet", string(secret4.Data["sshPrivateKey"]))

	return ctx
}

func assertSecretNotRemovedAfterRemovingFromPaas(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, "sshpaas", t, cfg)
	paas.Spec.SshSecrets = nil
	paas.Spec.Capabilities.SSO.SshSecrets = nil

	if err := updatePaasSync(ctx, cfg, paas); err != nil {
		t.Fatal(err)
	}

	// Wait for reconciliation of sso paasns
	ssopaasns := getOrFail(ctx, "sso", "sshpaas", &api.PaasNS{}, t, cfg)
	// TODO (portly-halicore-76) wait better for this but how?
	time.Sleep(10 * time.Second)
	require.NoError(t, waitForPaasNSReconciliation(ctx, cfg, ssopaasns), "SSO PaasNS reconciliation succeeds")

	secrets := &corev1.SecretList{}
	err := cfg.Client().Resources().List(ctx, secrets, func(opts *v1.ListOptions) { opts.FieldSelector = "metadata.namespace=sshpaas-sso" })
	require.NoError(t, err)
	assert.Len(t, secrets.Items, 4)

	// Assert each secret
	secret1 := getOrFail(ctx, "paas-ssh-1deb30f1", "sshpaas-sso", &corev1.Secret{}, t, cfg)
	assert.NotEmpty(t, secret1)
	assert.Equal(t, paas.UID, secret1.OwnerReferences[0].UID)
	assert.Equal(t, "repo-creds", secret1.Labels["argocd.argoproj.io/secret-type"])
	assert.Equal(t, "git", string(secret1.Data["type"]))
	assert.Equal(t, "ssh://git@scm/some-repo.git", string(secret1.Data["url"]))
	assert.Equal(t, "updatet", string(secret1.Data["sshPrivateKey"]))

	secret2 := getOrFail(ctx, "paas-ssh-5c51424e", "sshpaas-sso", &corev1.Secret{}, t, cfg)
	assert.NotEmpty(t, secret2)
	assert.Equal(t, paas.UID, secret2.OwnerReferences[0].UID)
	assert.Equal(t, "repo-creds", secret2.Labels["argocd.argoproj.io/secret-type"])
	assert.Equal(t, "git", string(secret2.Data["type"]))
	assert.Equal(t, "ssh://git@scm/some-other-repo.git", string(secret2.Data["url"]))
	assert.Equal(t, "updatet", string(secret2.Data["sshPrivateKey"]))

	secret3 := getOrFail(ctx, "paas-ssh-6df19938", "sshpaas-sso", &corev1.Secret{}, t, cfg)
	assert.NotEmpty(t, secret3)
	assert.Equal(t, paas.UID, secret3.OwnerReferences[0].UID)
	assert.Equal(t, "repo-creds", secret3.Labels["argocd.argoproj.io/secret-type"])
	assert.Equal(t, "git", string(secret3.Data["type"]))
	assert.Equal(t, "ssh://git@scm/some-second-repo.git", string(secret3.Data["url"]))
	assert.Equal(t, "updatet", string(secret3.Data["sshPrivateKey"]))

	secret4 := getOrFail(ctx, "paas-ssh-c1e4bede", "sshpaas-sso", &corev1.Secret{}, t, cfg)
	assert.NotEmpty(t, secret4)
	assert.Equal(t, paas.UID, secret4.OwnerReferences[0].UID)
	assert.Equal(t, "repo-creds", secret4.Labels["argocd.argoproj.io/secret-type"])
	assert.Equal(t, "git", string(secret4.Data["type"]))
	assert.Equal(t, "ssh://git@scm/some-other-second-repo.git", string(secret4.Data["url"]))
	assert.Equal(t, "updatet", string(secret4.Data["sshPrivateKey"]))

	return ctx
}
