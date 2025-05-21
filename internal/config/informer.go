package config

import (
	"context"
	"fmt"
	"reflect"

	"github.com/rs/zerolog/log"

	"k8s.io/client-go/tools/cache"
	cache2 "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
)

type configInformer struct {
	mgr manager.Manager
}

// SetupPaasConfigInformer will add an informer to the manager and inform on PaasConfig changes
func SetupPaasConfigInformer(mgr manager.Manager) error {
	return mgr.Add(&configInformer{mgr: mgr})
}

// Start is the runnable for the PaasConfigInformer
func (w *configInformer) Start(ctx context.Context) error {
	log.Info().Msg("starting config informer")

	informer, err := w.mgr.GetCache().GetInformer(ctx, &v1alpha1.PaasConfig{})
	if err != nil {
		return fmt.Errorf("failed to get informer for PaasConfig: %w", err)
	}

	err = addPaasConfigEventHandler(informer)
	if err != nil {
		return fmt.Errorf("failed to add event handler: %w", err)
	}

	<-ctx.Done() // Keep the goroutine alive
	return nil
}

// addPaasConfigEventHandler adds the update handler to the informer. We're not interested in Additions and Deletions
func addPaasConfigEventHandler(informer cache2.Informer) error {
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: updateHandler,
	})
	return err
}

// updateHandler determines whether the updated config is the active one and if there are spec changes to consume.
func updateHandler(_, newObj interface{}) {
	cfg, ok := newObj.(*v1alpha1.PaasConfig)
	if !ok {
		return
	}
	if cfg.IsActive() && !reflect.DeepEqual(cfg.Spec, GetConfig().Spec) {
		log.Info().Msg("updating config")
		SetConfig(*cfg)
	} else {
		log.Debug().Msg("config not changed")
	}
}

func (w *configInformer) NeedLeaderElection() bool {
	return false // All replicas need to do this even though they might not be a leader
}
