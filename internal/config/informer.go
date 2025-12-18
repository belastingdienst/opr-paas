/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package config

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/belastingdienst/opr-paas/v4/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v4/internal/logging"
)

type configInformer struct {
	mgr manager.Manager
}

// SetupPaasConfigInformer will add an informer to the manager and inform on PaasConfig changes
func SetupPaasConfigInformer(mgr manager.Manager) error {
	// Adds informer for PaasConfig to force the cache to sync
	_, err := mgr.GetCache().GetInformer(context.Background(), &v1alpha2.PaasConfig{})
	if err != nil {
		return err
	}
	return mgr.Add(&configInformer{mgr: mgr})
}

// Start is the runnable for the PaasConfigInformer
func (w *configInformer) Start(ctx context.Context) error {
	ctx, logger := logging.GetLogComponent(ctx, logging.ConfigComponent)
	logger.Info().Msg("starting config informer")

	<-ctx.Done() // Keep the goroutine alive
	return nil
}

func (w *configInformer) NeedLeaderElection() bool {
	// Returning false means that this runnable does not need LeaderElection
	return false // All replicas need to do this even though they might not be a leader
}

// GetCache satisfies hasCache interface so the manager knows to put this runnable in the cache group
func (w *configInformer) GetCache() cache.Cache {
	return w.mgr.GetCache()
}
