package config

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/logging"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Watch will open a new connection to Kubernetes and watch for PaasConfig changes
func Watch(restConfig *rest.Config, httpClient *http.Client, scheme *runtime.Scheme) error {
	ctx := context.Background()
	ctx, logger := logging.SetWatcherLogger(ctx, &v1alpha1.PaasConfig{})
	withWatchClient, err := client.NewWithWatch(
		restConfig,
		client.Options{
			HTTPClient: httpClient,
			Scheme:     scheme,
		},
	)
	if err != nil {
		logger.Panic().Msg(fmt.Errorf("failed to get watcher client: %w", err).Error())
	}
	watcher, err := withWatchClient.Watch(ctx, &v1alpha1.PaasConfigList{}, &client.ListOptions{})
	if err != nil {
		logger.Panic().Msg(fmt.Errorf("failed watch for PaasConfig changes: %w", err).Error())
	}
	if err != nil {
		return err
	}

	for {
		event := <-watcher.ResultChan()
		paasConfig, ok := event.Object.(*v1alpha1.PaasConfig)
		if !ok {
			return errors.New("unexpected object type in PaasConfig watcher")
		}
		switch event.Type {
		case watch.Added, watch.Modified:
			logger.Info().Msg("setting PaasConfig")
			SetConfig(*paasConfig)
		case watch.Deleted:
			logger.Info().Msg("resetting PaasConfig")
			SetConfig(v1alpha1.PaasConfig{})
		}
	}
}
