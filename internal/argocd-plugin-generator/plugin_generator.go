/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package argocd_plugin_generator

import (
	"context"
	"fmt"
	"os"

	"github.com/belastingdienst/opr-paas/v4/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v4/internal/logging"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

const tokenEnvVar = "ARGOCD_GENERATOR_TOKEN"

// GeneratorServerInterface defines the contract for a plug-in generator server.
//
// Any server implementation must be able to start serving requests
// and indicate whether it requires leader election before running.
type GeneratorServerInterface interface {
	// Start launches the server and begins handling incoming requests
	// until the given context is canceled or an error occurs.
	Start(ctx context.Context) error

	// StartedChecker returns a healthz.Checker which is healthy after the
	// server has been started.
	StartedChecker() healthz.Checker
}

// PluginGenerator ties together the plug-in generator's HTTP server
// and business logic service.
//
// It is responsible for wiring the service (which interacts with
// Kubernetes resources) to the server (which handles incoming requests),
// and can be added to the controller-runtime manager.
type PluginGenerator struct {
	service *Service
	server  GeneratorServerInterface
	cache   cache.Cache
}

// New creates a new PluginGenerator instance using the provided
// controller-runtime Client.
//
// The client is passed to the Service for interacting with Kubernetes
// objects, and the server will be configured internally to use this service.
func New(kclient client.Client, c cache.Cache, bindAddr string) (*PluginGenerator, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, logger := logging.GetLogComponent(ctx, logging.PluginGeneratorComponent)
	generatorService := NewService(kclient)

	token := os.Getenv(tokenEnvVar)
	logger.Debug().Str("token", token).Msg("token")
	handler := NewHandler(generatorService, token)

	server := NewServer(ServerOptions{
		Addr:        bindAddr,
		TokenEnvVar: tokenEnvVar,
	}, handler)

	// Ensure informer for Paas exists, to trigger sync of the cache
	if _, err := c.GetInformer(context.Background(), &v1alpha2.Paas{}); err != nil {
		return nil, fmt.Errorf("failed to get informer for Paas: %w", err)
	}

	// Ensure informer for PaasConfig exists, to trigger sync of the cache
	if _, err := c.GetInformer(context.Background(), &v1alpha2.PaasConfig{}); err != nil {
		return nil, fmt.Errorf("failed to get informer for Paas: %w", err)
	}

	logger.Debug().Msg("New PluginGenerator")
	return &PluginGenerator{
		service: generatorService,
		server:  server,
		cache:   c,
	}, nil
}

// Start satisfies Runnable so that the manager can start the runnable
func (pg *PluginGenerator) Start(ctx context.Context) error {
	_, logger := logging.GetLogComponent(ctx, logging.PluginGeneratorComponent)
	logger.Debug().Msg("started")
	return pg.server.Start(ctx)
}

// NeedLeaderElection satisfies LeaderElectionRunnable
func (pg *PluginGenerator) NeedLeaderElection() bool {
	// Returning false means that this runnable does not need LeaderElection
	return false
}

// StartedChecker return a health checker
func (pg *PluginGenerator) StartedChecker() healthz.Checker {
	return pg.server.StartedChecker()
}

// GetCache satisfies hasCache interface so the manager knows to put this runnable in the cache group
func (pg *PluginGenerator) GetCache() cache.Cache {
	return pg.cache
}
