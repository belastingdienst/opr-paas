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

type serverOptions struct {
	Addr        string
	TokenEnvVar string
}

type generatorServer struct {
	opts    serverOptions
	handler http.Handler
	server  *http.Server
}

// NewServer returns a new generatorServer based on the serverOptions and the http.Handler
func NewServer(opts serverOptions, handler http.Handler) *generatorServer {
	return &generatorServer{
		opts:    opts,
		handler: handler,
	}
}

// Start starts the generatorServer. If it fails, it returns an error.
func (s *generatorServer) Start(ctx context.Context) error {
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

// StartedChecker returns a healthz.Checker which is healthy after the
// server has been started.
// func (s *generatorServer) StartedChecker() healthz.Checker {
//	config := &tls.Config{
//		InsecureSkipVerify: true,
//	}
//	return func(req *http.Request) error {
//
//		if !s.started {
//			return fmt.Errorf("webhook server has not been started yet")
//		}
//
//		d := &net.Dialer{Timeout: 10 * time.Second}
//		conn, err := tls.DialWithDialer(d, "tcp", s.opts.Addr, config)
//		if err != nil {
//			return fmt.Errorf("webhook server is not reachable: %w", err)
//		}
//
//		if err := conn.Close(); err != nil {
//			return fmt.Errorf("webhook server is not reachable: closing connection: %w", err)
//		}
//
//		return nil
//	}
// }

func (s *generatorServer) NeedLeaderElection() bool {
	return false
}
