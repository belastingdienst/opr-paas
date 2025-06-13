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
	"errors"
	"fmt"
	appv1 "github.com/belastingdienst/opr-paas/v2/internal/stubs/argoproj/v1alpha1"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/go-logr/zerologr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	quotav1 "github.com/openshift/api/quota/v1"
	userv1 "github.com/openshift/api/user/v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/belastingdienst/opr-paas-crypttool/pkg/crypt"
	"github.com/belastingdienst/opr-paas/v2/api/v1alpha2"
	// +kubebuilder:scaffold:imports
)

var (
	cfg           *rest.Config
	k8sClient     client.Client
	testEnv       *envtest.Environment
	pubkey        *rsa.PublicKey
	genericConfig = v1alpha2.PaasConfig{
		Spec: v1alpha2.PaasConfigSpec{
			DecryptKeysSecret: v1alpha2.NamespacedName{
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

	err = v1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = userv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = quotav1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = appv1.AddToScheme(scheme.Scheme)
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

func patchAppSet(ctx context.Context, newAppSet *appv1.ApplicationSet) {
	oldAppSet := &appv1.ApplicationSet{}
	namespacedName := types.NamespacedName{
		Name:      newAppSet.Name,
		Namespace: newAppSet.Namespace,
	}
	err := k8sClient.Get(ctx, namespacedName, oldAppSet)
	if err == nil {
		// Patch
		patch := client.MergeFrom(oldAppSet.DeepCopy())
		oldAppSet.Spec = newAppSet.Spec
		err = k8sClient.Patch(ctx, oldAppSet, patch)
		Expect(err).NotTo(HaveOccurred())
	} else {
		Expect(err.Error()).To(MatchRegexp(`applicationsets.argoproj.io .* not found`))
		err = k8sClient.Create(ctx, newAppSet)
		Expect(err).NotTo(HaveOccurred())
	}
}

func createPaasPrivateKeySecret(ctx context.Context, ns string, name string, privateKey []byte) {
	// Set up private key
	err := k8sClient.Create(ctx, &corev1.Secret{
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

func newGeneratedCrypt(ctx string) (myCrypt *crypt.Crypt, privateKey []byte, err error) {
	tmpFileError := "failed to get new tmp private key file: %w"
	privateKeyFile, err := os.CreateTemp("", "private")
	if err != nil {
		return nil, nil, fmt.Errorf(tmpFileError, err)
	}
	publicKeyFile, err := os.CreateTemp("", "public")
	if err != nil {
		return nil, nil, fmt.Errorf(tmpFileError, err)
	}
	myCrypt, err = crypt.NewGeneratedCrypt(privateKeyFile.Name(), publicKeyFile.Name(), ctx)
	if err != nil {
		return nil, nil, fmt.Errorf(tmpFileError, err)
	}
	privateKey, err = os.ReadFile(privateKeyFile.Name())
	if err != nil {
		return nil, nil, errors.New("failed to read private key from file")
	}

	return myCrypt, privateKey, nil
}

func assureNamespace(ctx context.Context, namespaceName string) {
	oldNs := &corev1.Namespace{}
	namespacedName := types.NamespacedName{
		Name: namespaceName,
	}
	err := k8sClient.Get(ctx, namespacedName, oldNs)
	if err == nil {
		return
	}
	Expect(err.Error()).To(MatchRegexp(`namespaces .* not found`))
	err = k8sClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespaceName},
	})
	Expect(err).NotTo(HaveOccurred())
}

func assureNamespaceWithPaasReference(ctx context.Context, namespaceName string, paasName string) {
	assureNamespace(ctx, namespaceName)
	paas := &v1alpha2.Paas{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: paasName}, paas)
	Expect(err).NotTo(HaveOccurred())
	ns := &corev1.Namespace{}
	err = k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, ns)
	Expect(err).NotTo(HaveOccurred())

	if !paas.AmIOwner(ns.GetOwnerReferences()) {
		patchedNs := client.MergeFrom(ns.DeepCopy())
		controllerutil.SetControllerReference(paas, ns, scheme.Scheme)
		err = k8sClient.Patch(ctx, ns, patchedNs)
		Expect(err).NotTo(HaveOccurred())
	}
}

func assurePaas(ctx context.Context, newPaas v1alpha2.Paas) {
	oldPaas := &v1alpha2.Paas{}
	namespacedName := types.NamespacedName{
		Name: newPaas.Name,
	}
	err := k8sClient.Get(ctx, namespacedName, oldPaas)
	if err == nil {
		return
	}
	Expect(err.Error()).To(MatchRegexp(`paas.cpet.belastingdienst.nl .* not found`))
	err = k8sClient.Create(ctx, &newPaas)
	Expect(err).NotTo(HaveOccurred())
}

func validatePaasNSExists(ctx context.Context, namespaceName string, paasNSName string) {
	pns := v1alpha2.PaasNS{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: paasNSName, Namespace: namespaceName}, &pns)
	Expect(err).NotTo(HaveOccurred())
}

func assurePaasNS(ctx context.Context, paasNs v1alpha2.PaasNS) {
	assureNamespace(ctx, paasNs.GetNamespace())
	oldPaasNS := &v1alpha2.PaasNS{}
	namespacedName := types.NamespacedName{Name: paasNs.GetName(), Namespace: paasNs.GetNamespace()}
	err := k8sClient.Get(ctx, namespacedName, oldPaasNS)
	if err == nil {
		return
	}
	Expect(err.Error()).To(MatchRegexp(`paasns.cpet.belastingdienst.nl .* not found`))
	if paasNs.Spec.Paas == "" {
		paasNs.Spec.Paas = paasNs.GetNamespace()
	}
	err = k8sClient.Create(ctx, &paasNs)
	Expect(err).NotTo(HaveOccurred())
}

func getPaas(ctx context.Context, paasName string) *v1alpha2.Paas {
	paas := &v1alpha2.Paas{}
	namespacedName := types.NamespacedName{
		Name: paasName,
	}
	err := k8sClient.Get(ctx, namespacedName, paas)
	Expect(err).NotTo(HaveOccurred())
	Expect(paas).NotTo(BeNil())
	return paas
}

func assureAppSet(ctx context.Context, name string, namespace string) {
	appSet := &appv1.ApplicationSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appv1.ApplicationSetSpec{
			Generators: []appv1.ApplicationSetGenerator{},
		},
	}
	err := k8sClient.Create(ctx, appSet)
	Expect(err).NotTo(HaveOccurred())
}
