/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package argocd_plugin_generator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/belastingdienst/opr-paas-cli/v2/pkg/crypt"
	"github.com/belastingdienst/opr-paas/v5/api/v1alpha2"
	"github.com/go-logr/zerologr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
)

func TestArgoCDPluginGenerator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GeneratorServer Suite")
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

	err = v1alpha2.AddToScheme(testEnv.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: testEnv.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func createNamespace(ctx context.Context, cl client.Client, ns string) {
	// Create system namespace
	err := cl.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	})
	if err != nil {
		Fail(fmt.Errorf("failed to create %s namespace: %w", ns, err).Error())
	}
}

func createPaasPrivateKeySecret(ctx context.Context, cl client.Client, ns string, name string, privateKey []byte) {
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

func newGeneratedCrypt(cryptContext string) (myCrypt *crypt.Crypt, privateKey []byte, err error) {
	tmpFileError := "failed to get new tmp private key file: %w"
	privateKeyFile, err := os.CreateTemp("", "private")
	if err != nil {
		return nil, nil, fmt.Errorf(tmpFileError, err)
	}
	publicKeyFile, err := os.CreateTemp("", "public")
	if err != nil {
		return nil, nil, fmt.Errorf(tmpFileError, err)
	}
	myCrypt, err = crypt.NewGeneratedCrypt(privateKeyFile.Name(), publicKeyFile.Name(), cryptContext)
	if err != nil {
		return nil, nil, fmt.Errorf(tmpFileError, err)
	}
	privateKey, err = os.ReadFile(privateKeyFile.Name())
	if err != nil {
		return nil, nil, errors.New("failed to read private key from file")
	}

	return myCrypt, privateKey, nil
}
