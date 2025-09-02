package config

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"k8s.io/client-go/tools/cache"
	cache2 "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v3/internal/logging"
)

type configInformer struct {
	mgr manager.Manager
}

// SetupPaasConfigInformer will add an informer to the manager and inform on PaasConfig changes
func SetupPaasConfigInformer(mgr manager.Manager) error {
	return mgr.Add(&configInformer{mgr: mgr})
}

func (w *configInformer) setInitialConfig(ctx context.Context) error {
	var list v1alpha2.PaasConfigList

	if err := w.mgr.GetClient().List(ctx, &list); err != nil {
		return fmt.Errorf("failed to retrieve PaasConfigs: %w", err)
	}

	switch len(list.Items) {
	case 0:
		SetConfig(v1alpha2.PaasConfig{})
	case 1:
		SetConfig(list.Items[0])
	default:
		return errors.New("more than one PaasConfig, this should not happen")
	}
	return nil
}

// Start is the runnable for the PaasConfigInformer
func (w *configInformer) Start(ctx context.Context) error {
	ctx, logger := logging.GetLogComponent(ctx, "config_watcher")
	logger.Info().Msg("starting config informer")

	logger.Debug().Msg("setting initial paasConfig definition (empty when no PaasConfig is loaded)")
	if err := w.setInitialConfig(ctx); err != nil {
		logger.Error().Msgf("error setting initial config: %e", err)
		return err
	}

	informer, err := w.mgr.GetCache().GetInformer(ctx, &v1alpha2.PaasConfig{})
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
	cfg, ok := newObj.(*v1alpha2.PaasConfig)
	if !ok {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, logger := logging.GetLogComponent(ctx, "config_watcher")
	if cfg.IsActive() && !reflect.DeepEqual(cfg.Spec, GetConfig().Spec) {
		logger.Info().Msg("updating config")
		SetConfig(*cfg)
	} else {
		logger.Debug().Msg("config not changed")
	}
}

func (w *configInformer) NeedLeaderElection() bool {
	// Returning false means that this runnable does not need LeaderElection
	return false // All replicas need to do this even though they might not be a leader
}
