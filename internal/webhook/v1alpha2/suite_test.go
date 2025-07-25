/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha2

// Excuse Ginkgo use from revive errors
//revive:disable:dot-imports

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/v3/api/v1alpha2"
	"github.com/go-logr/zerologr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cancel               context.CancelFunc
	cfg                  *rest.Config
	ctx                  context.Context
	k8sClient            client.Client
	testEnv              *envtest.Environment
	paasConfigSystem     = "paasconfig-testns"
	paasConfigPkSecret   = "paasconfig-testpksecret"
	paasConfigPrivateKey []byte
)

func TestWebhooks(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Webhook Suite")
}

var _ = BeforeSuite(func() {
	log.Logger = log.Level(zerolog.DebugLevel).
		Output(zerolog.ConsoleWriter{Out: GinkgoWriter})
	ctrl.SetLogger(zerologr.New(&log.Logger))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	binDirs, _ := filepath.Glob(filepath.Join("..", "..", "..", "bin", "k8s",
		fmt.Sprintf("*-%s-%s", runtime.GOOS, runtime.GOARCH)))
	slices.Sort(binDirs)
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "manifests", "crd", "bases")},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: binDirs[len(binDirs)-1],

		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "..", "manifests", "webhook")},
		},
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = v1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = v1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// start webhook server using Manager.
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    webhookInstallOptions.LocalServingHost,
			Port:    webhookInstallOptions.LocalServingPort,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),
		LeaderElection: false,
		Metrics:        metricsserver.Options{BindAddress: "0"},
	})
	Expect(err).NotTo(HaveOccurred())

	err = SetupPaasWebhookWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	// Ensure we have a namespace and privatekey for PaasConfig testing
	createNamespace(k8sClient, paasConfigSystem)
	createPaasPrivateKeySecret(k8sClient, paasConfigSystem, paasConfigPkSecret, paasConfigPrivateKey)

	// +kubebuilder:scaffold:webhook

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	// wait for the webhook server to get ready.
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	Eventually(func() error {
		var conn *tls.Conn
		conn, err = tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true}) //nolint:gosec
		if err != nil {
			return err
		}

		return conn.Close()
	}).Should(Succeed())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func createNamespace(cl client.Client, ns string) {
	// Create system namespace
	err := cl.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	})
	if err != nil {
		Fail(fmt.Errorf("failed to create %s namespace: %w", ns, err).Error())
	}
}

func createPaasPrivateKeySecret(cl client.Client, ns string, name string, privateKey []byte) {
	// Set up private key
	err := cl.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Data: map[string][]byte{"privatekey0": privateKey},
	})
	if err != nil {
		Fail(fmt.Errorf("failed to create %s.%s secret: %w", ns, name, err).Error())
	}
}

func createPaasNamespace(cl client.Client, paas v1alpha2.Paas, nsName string) {
	controller := true
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       paas.Name,
					APIVersion: v1alpha2.GroupVersion.Version,
					Kind:       "Paas",
					UID:        paas.UID,
					Controller: &controller,
				},
			},
		},
	}
	err := cl.Create(ctx, ns)
	Expect(err).NotTo(HaveOccurred())
}
