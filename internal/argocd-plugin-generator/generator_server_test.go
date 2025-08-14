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

var _ = Describe("GeneratorServer", func() {
	var (
		server          *GeneratorServer
		opts            ServerOptions
		handler         http.Handler
		ctx             context.Context
		cancel          context.CancelFunc
		addr            string
		testTokenEnvVar string
		tokenValue      string
	)

	const randomAddress = "127.0.0.1:0"

	BeforeEach(func() {
		addr = randomAddress // let OS choose a free port
		testTokenEnvVar = "GENERATOR_TOKEN"
		tokenValue = "supersecrettoken"

		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("hello"))
		})

		opts = ServerOptions{
			Addr:        addr,
			TokenEnvVar: testTokenEnvVar,
		}

		ctx, cancel = context.WithCancel(context.Background())
	})

	AfterEach(func() {
		cancel()
		os.Unsetenv(testTokenEnvVar)
	})

	It("returns an error if the token environment variable is not set", func() {
		server = NewServer(opts, handler)
		err := server.Start(ctx)
		Expect(err).To(MatchError(fmt.Sprintf("environment variable %s not set", testTokenEnvVar)))
	})

	It("starts the server successfully when the token is set", func() {
		os.Setenv(testTokenEnvVar, tokenValue)
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
		os.Setenv(testTokenEnvVar, tokenValue)
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
		os.Setenv(testTokenEnvVar, tokenValue)

		// First listener to occupy the port
		ln, err := net.Listen("tcp", randomAddress)
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

	It("StartedChecker returns no error after server has started", func() {
		os.Setenv(testTokenEnvVar, tokenValue)

		// Create a listener to get a free port
		ln, err := net.Listen("tcp", randomAddress)
		Expect(err).ToNot(HaveOccurred())
		addrInUse := ln.Addr().String()
		_ = ln.Close() // release the port so the server can bind to it

		opts.Addr = addrInUse
		server = NewServer(opts, handler)

		done := make(chan error)
		go func() {
			done <- server.Start(ctx)
		}()

		time.Sleep(200 * time.Millisecond) // give server time to start

		checker := server.StartedChecker()
		req := &http.Request{} // dummy request

		Eventually(func() error {
			return checker(req)
		}, 2*time.Second, 100*time.Millisecond).Should(Succeed())

		cancel()
		<-done
	})
})
