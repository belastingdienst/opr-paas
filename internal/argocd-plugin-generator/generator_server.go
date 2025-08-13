/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package argocd_plugin_generator

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

const pluginServerTimeout = 10 * time.Second

// ServerOptions contains configuration values for starting the plug-in generator HTTP server.
//
// Addr specifies the TCP address for the server to listen on.
// For example, ":4355" will bind on all interfaces at port 4355.
// Use "0" to disable the server entirely.
//
// TokenEnvVar specifies the name of the environment variable from which
// the bearer token will be read. This token is required for authenticating
// incoming requests from ArgoCD.
type ServerOptions struct {
	Addr        string
	TokenEnvVar string
}

// GeneratorServer represents the HTTP server that exposes the ArgoCD plug-in generator endpoint.
//
// It is responsible for starting, stopping, and serving HTTP requests
// using the provided handler. The server is typically added to a
// controller-runtime Manager so it can run alongside the main operator.
//
// Fields:
//   - opts: Holds the server configuration (address, token environment variable).
//   - handler: The HTTP handler that processes incoming plug-in generator requests.
//   - server: The underlying *http.Server used for network communication.
type GeneratorServer struct {
	opts    ServerOptions
	handler http.Handler
	server  *http.Server
}

// NewServer returns a new GeneratorServer based on the ServerOptions and the http.Handler
func NewServer(opts ServerOptions, handler http.Handler) *GeneratorServer {
	return &GeneratorServer{
		opts:    opts,
		handler: handler,
	}
}

// Start starts the GeneratorServer. If it fails, it returns an error.
func (s *GeneratorServer) Start(ctx context.Context) error {
	token := os.Getenv(s.opts.TokenEnvVar)
	if token == "" {
		return fmt.Errorf("environment variable %s not set", s.opts.TokenEnvVar)
	}

	s.server = &http.Server{
		Addr:         s.opts.Addr,
		Handler:      s.handler,
		ReadTimeout:  pluginServerTimeout,
		WriteTimeout: pluginServerTimeout,
	}

	ln, err := net.Listen("tcp", s.opts.Addr)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_ = s.server.Shutdown(shutdownCtx)
	}()

	return s.server.Serve(ln)
}
