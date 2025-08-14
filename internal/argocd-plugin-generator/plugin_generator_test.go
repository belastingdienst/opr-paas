/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package argocd_plugin_generator

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

// mockGeneratorServer implements only what PluginGenerator needs
type mockGeneratorServer struct {
	startCalled bool
	startErr    error
	needLeader  bool
}

func (m *mockGeneratorServer) Start(ctx context.Context) error {
	m.startCalled = true
	return m.startErr
}

func (m *mockGeneratorServer) StartedChecker() healthz.Checker {
	// To implement the interface, we return a started Checker
	return func(req *http.Request) error {
		return nil
	}
}

func (m *mockGeneratorServer) NeedLeaderElection() bool {
	return m.needLeader
}

var _ = Describe("PluginGenerator", func() {
	var (
		mockServer *mockGeneratorServer
		pg         *PluginGenerator
	)

	BeforeEach(func() {
		mockServer = &mockGeneratorServer{}
		pg = &PluginGenerator{
			service: nil, // Not relevant in these tests
			server:  mockServer,
		}
	})

	AfterEach(func() {
		_ = os.Unsetenv("ARGOCD_GENERATOR_TOKEN")
	})

	Context("New", func() {
		It("should create a PluginGenerator with initialized Service and server", func() {
			fakeClient := fake.NewClientBuilder().Build()

			_ = os.Setenv("ARGOCD_GENERATOR_TOKEN", "test-token")
			defer os.Unsetenv("ARGOCD_GENERATOR_TOKEN")

			pg = New(fakeClient, ":4355")

			Expect(pg).ToNot(BeNil())
			Expect(pg.service).ToNot(BeNil())
			Expect(pg.server).ToNot(BeNil())
		})
	})

	Context("Start", func() {
		It("should call Start on the server", func() {
			mockServer.startErr = nil

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()

			err := pg.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(mockServer.startCalled).To(BeTrue())
		})

		It("should return an error if server.Start fails", func() {
			mockServer.startErr = errors.New("start failed")

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()

			err := pg.Start(ctx)
			Expect(err).To(MatchError("start failed"))
			Expect(mockServer.startCalled).To(BeTrue())
		})
	})
})
