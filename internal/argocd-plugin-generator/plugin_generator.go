/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package argocd_plugin_generator

import (
	"context"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/client"
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
}

// New creates a new PluginGenerator instance using the provided
// controller-runtime Client.
//
// The client is passed to the Service for interacting with Kubernetes
// objects, and the server will be configured internally to use this service.
func New(kclient client.Client, bindAddr string) *PluginGenerator {
	generatorService := NewService(kclient)

	token := os.Getenv(tokenEnvVar)
	handler := NewHandler(generatorService, token)

	server := NewServer(ServerOptions{
		Addr:        bindAddr,
		TokenEnvVar: tokenEnvVar,
	}, handler)

	return &PluginGenerator{
		service: generatorService,
		server:  server,
	}
}

// Start satisfies Runnable so that the manager can start the runnable
func (pg *PluginGenerator) Start(ctx context.Context) error {
	return pg.server.Start(ctx)
}

// NeedLeaderElection satisfies LeaderElectionRunnable
func (pg *PluginGenerator) NeedLeaderElection() bool {
	return false
}
