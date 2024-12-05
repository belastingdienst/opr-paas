package e2e

import (
	"context"
	"testing"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/quota"
	argo "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"

	quotav1 "github.com/openshift/api/quota/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	paasWithCapability6 = "cap6paas"
	cap6Name            = "cap6"
	paasCap6            = "cap6paas-cap6"
	cap6ApplicationSet  = "cap6as"
	cap6StatusMessage   = "Capability 'cap6' is not configured"
)

func TestCapabilityCap6(t *testing.T) {
	paasSpec := api.PaasSpec{
		Requestor: "paas-user",
		Quota:     make(quota.Quota),
		Capabilities: api.PaasCapabilities{
			cap6Name: api.PaasCapability{Enabled: true},
		},
	}

	testenv.Test(
		t,
		features.New("Capability 6").
			Setup(createPaasFn(paasWithCapability6, paasSpec)).
			Assess("is created", assertCap6NotCreated).
			Teardown(teardownPaasFn(paasWithCapability6)).
			Feature(),
	)
}

func assertCap6NotCreated(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	paas := getPaas(ctx, paasWithCapability6, t, cfg)
	namespace := getOrFail(ctx, paasWithCapability6, cfg.Namespace(), &corev1.Namespace{}, t, cfg)
	cap6Quota := getOrFail(ctx, paasCap6, cfg.Namespace(), &quotav1.ClusterResourceQuota{}, t, cfg)
	getAndFail(ctx, cap6ApplicationSet, applicationSetNamespace, &argo.ApplicationSet{}, t, cfg)
	getAndFail(ctx, paasCap6, cfg.Namespace(), &corev1.Namespace{}, t, cfg)

	// Paas has correct name
	assert.Equal(t, paasWithCapability6, paas.Name)

	// Paas Namespace exist
	assert.Equal(t, paasWithCapability6, namespace.Name)

	// Quota exists
	assert.Equal(t, paasCap6, cap6Quota.Name)

	// Paas has status message
	assert.Contains(t, paas.Status.Messages, cap6StatusMessage)

	return ctx
}
