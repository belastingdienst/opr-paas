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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("generatorServer", func() {
	var (
		server      *generatorServer
		opts        serverOptions
		handler     http.Handler
		ctx         context.Context
		cancel      context.CancelFunc
		addr        string
		tokenEnvVar string
		tokenValue  string
	)

	BeforeEach(func() {
		addr = "127.0.0.1:0" // let OS choose a free port
		tokenEnvVar = "GENERATOR_TOKEN"
		tokenValue = "supersecrettoken"

		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("hello"))
		})

		opts = serverOptions{
			Addr:        addr,
			TokenEnvVar: tokenEnvVar,
		}

		ctx, cancel = context.WithCancel(context.Background())
	})

	AfterEach(func() {
		cancel()
		os.Unsetenv(tokenEnvVar)
	})

	It("returns an error if the token environment variable is not set", func() {
		server = NewServer(opts, handler)
		err := server.Start(ctx)
		Expect(err).To(MatchError(fmt.Sprintf("environment variable %s not set", tokenEnvVar)))
	})

	It("starts the server successfully when the token is set", func() {
		os.Setenv(tokenEnvVar, tokenValue)
		server = NewServer(opts, handler)

		go func() {
			// cancel context after short time to stop the server
			time.Sleep(500 * time.Millisecond)
			cancel()
		}()

		err := server.Start(ctx)
		Expect(err).To(SatisfyAny(BeNil(), MatchError("http: Server closed")))
	})

	It("shuts down the server when context is cancelled", func() {
		os.Setenv(tokenEnvVar, tokenValue)
		server = NewServer(opts, handler)

		done := make(chan error)
		go func() {
			done <- server.Start(ctx)
		}()

		// give the server time to start
		time.Sleep(200 * time.Millisecond)

		// cancel the context to trigger shutdown
		cancel()

		select {
		case err := <-done:
			Expect(err).To(SatisfyAny(BeNil(), MatchError("http: Server closed")))
		case <-time.After(2 * time.Second):
			Fail("server did not shut down in time")
		}
	})

	It("returns an error if net.Listen fails (address already in use)", func() {
		os.Setenv(tokenEnvVar, tokenValue)

		// First listener to occupy the port
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).ToNot(HaveOccurred())
		defer ln.Close()

		// Use the exact same port to force net.Listen failure
		addrInUse := ln.Addr().String()
		opts.Addr = addrInUse

		server = NewServer(opts, handler)

		// Attempting to bind to the same address should fail
		err = server.Start(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("address already in use"))
	})
})
