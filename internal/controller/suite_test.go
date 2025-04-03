/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package controller

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	gitops "github.com/belastingdienst/opr-paas/internal/stubs/argoproj-labs/v1beta1"
	argocd "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"

	"github.com/go-logr/zerologr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	quotav1 "github.com/openshift/api/quota/v1"
	userv1 "github.com/openshift/api/user/v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/crypt"
	// +kubebuilder:scaffold:imports
)

var (
	cfg           *rest.Config
	k8sClient     client.Client
	testEnv       *envtest.Environment
	pubkey        *rsa.PublicKey
	genericConfig = api.PaasConfig{
		Spec: api.PaasConfigSpec{
			DecryptKeysSecret: api.NamespacedName{
				Name:      "keys",
				Namespace: "paas-system",
			},
		},
	}
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

func setupPaasSys() {
	ctx := context.Background()

	// Create system namespace
	err := k8sClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "paas-system"},
	})
	Expect(err).NotTo(HaveOccurred())

	// Create clusterwide argo namespace
	err = k8sClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "asns"},
	})
	Expect(err).NotTo(HaveOccurred())

	// Set up private key
	privkey, err := rsa.GenerateKey(rand.Reader, crypt.AESKeySize)
	Expect(err).NotTo(HaveOccurred())
	err = k8sClient.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keys",
			Namespace: "paas-system",
		},
		Data: map[string][]byte{
			"privatekey0": pem.EncodeToMemory(
				&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privkey)},
			),
		},
	})
	Expect(err).NotTo(HaveOccurred())

	// Save public key so we can encrypt things within tests
	pubkey = &privkey.PublicKey
}

var _ = BeforeSuite(func() {
	log.Logger = log.Level(zerolog.DebugLevel).
		Output(zerolog.ConsoleWriter{Out: GinkgoWriter})
	ctrl.SetLogger(zerologr.New(&log.Logger))

	By("bootstrapping test environment")
	binDirs, _ := filepath.Glob(filepath.Join("..", "..", "bin", "k8s",
		fmt.Sprintf("*-%s-%s", runtime.GOOS, runtime.GOARCH)))
	slices.Sort(binDirs)
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "manifests", "crd", "bases"),
			filepath.Join("..", "..", "test", "e2e", "manifests", "openshift"),
			filepath.Join("..", "..", "test", "e2e", "manifests", "gitops-operator"),
		},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: binDirs[len(binDirs)-1],
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = api.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = userv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// Add Argo to schema
	err = gitops.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = quotav1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = argocd.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	setupPaasSys()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
