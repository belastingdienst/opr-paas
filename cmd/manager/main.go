/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	argocdplugingenerator "github.com/belastingdienst/opr-paas/v4/internal/argocd-plugin-generator"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"

	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/belastingdienst/opr-paas/v4/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/v4/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v4/internal/config"
	"github.com/belastingdienst/opr-paas/v4/internal/controller"
	"github.com/belastingdienst/opr-paas/v4/internal/logging"
	"github.com/belastingdienst/opr-paas/v4/internal/version"
	webhookv1alpha1 "github.com/belastingdienst/opr-paas/v4/internal/webhook/v1alpha1"
	webhookv1alpha2 "github.com/belastingdienst/opr-paas/v4/internal/webhook/v1alpha2"
	"github.com/go-logr/zerologr"
	quotav1 "github.com/openshift/api/quota/v1"
	userv1 "github.com/openshift/api/user/v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	// +kubebuilder:scaffold:imports
)

const webhookErrMsg = "unable to create webhook"

var scheme = runtime.NewScheme()

type flags struct {
	metricsAddr                                      string
	enableLeaderElection                             bool
	secureMetrics                                    bool
	enableHTTP2                                      bool
	probeAddr                                        string
	getVersion                                       bool
	pretty                                           bool
	debug                                            bool
	componentDebugList                               string
	splitLogOutput                                   bool
	metricsCertPath, metricsCertName, metricsCertKey string
	webhookCertPath, webhookCertName, webhookCertKey string
	argocdPluginGenAddr                              string
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(quotav1.AddToScheme(scheme))
	utilruntime.Must(userv1.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(v1alpha2.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	f := configureFlags()
	configureLogging(f.pretty, f.debug, f.componentDebugList, f.splitLogOutput)

	if f.getVersion {
		fmt.Printf("opr-paas version %s", version.PaasVersion)
		os.Exit(0)
	}

	log.Info().Str("version", version.PaasVersion).Msg("opr-paas version")
	mgr := configureManager(f)

	log.Info().Msgf("starting manager version %s", version.PaasVersion)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Fatal().Err(err).Msg("problem starting manager")
	}
}

func configureFlags() *flags {
	f := &flags{}
	flag.StringVar(&f.metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&f.probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&f.getVersion, "version", false, "Print version and quit")
	flag.BoolVar(&f.enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&f.secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.StringVar(&f.webhookCertPath, "webhook-cert-path", "", "The directory that contains the webhook certificate.")
	flag.StringVar(&f.webhookCertName, "webhook-cert-name", "tls.crt", "The name of the webhook certificate file.")
	flag.StringVar(&f.webhookCertKey, "webhook-cert-key", "tls.key", "The name of the webhook key file.")
	flag.StringVar(&f.metricsCertPath, "metrics-cert-path", "",
		"The directory that contains the metrics server certificate.")
	flag.StringVar(&f.metricsCertName, "metrics-cert-name", "tls.crt",
		"The name of the metrics server certificate file.")
	flag.StringVar(&f.metricsCertKey, "metrics-cert-key", "tls.key", "The name of the metrics server key file.")
	flag.StringVar(&f.argocdPluginGenAddr, "argocd-plugin-generator-bind-address", "0", "The address the argocd plugin generator endpoint binds to. Use :4355 for HTTP, or leave as 0 to disable the argocd plugin generator service.") // nolint:revive
	flag.BoolVar(&f.enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	flag.BoolVar(&f.pretty, "pretty", false, "Pretty-print logging output")
	flag.BoolVar(&f.debug, "debug", false, "Log all debug messages")
	flag.StringVar(
		&f.componentDebugList,
		"component-debug",
		"",
		"Comma-separated list of components to log debug messages for.",
	)
	flag.BoolVar(&f.splitLogOutput, "split-log-output", false, "Send error logs to stderr, and the rest to stdout.")
	flag.Parse()

	return f
}

func configureLogging(pretty bool, debug bool, componentDebugList string, splitLogOutput bool) {
	var output io.Writer = os.Stderr

	if splitLogOutput {
		var errout io.Writer = os.Stderr
		var infout io.Writer = os.Stdout

		if pretty {
			errout = zerolog.ConsoleWriter{Out: errout}
			infout = zerolog.ConsoleWriter{Out: infout}
		}

		errw := zerolog.FilteredLevelWriter{
			Writer: zerolog.LevelWriterAdapter{Writer: errout},
			Level:  zerolog.ErrorLevel,
		}
		infw := infoLevelWriter{infout}
		output = zerolog.MultiLevelWriter(&errw, infw)
	} else if pretty {
		output = zerolog.ConsoleWriter{Out: output}
	}

	log.Logger = log.Output(output)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	logging.SetStaticLoggingConfig(debug, logging.NewComponentsFromString(componentDebugList))
	_, logger := logging.GetLogComponent(context.TODO(), logging.RuntimeComponent)
	ctrl.SetLogger(zerologr.New(logger))
}

type infoLevelWriter struct {
	io.Writer
}

func (w infoLevelWriter) Write(p []byte) (int, error) {
	return w.Writer.Write(p)
}

func (w infoLevelWriter) WriteLevel(level zerolog.Level, p []byte) (int, error) {
	if level <= zerolog.InfoLevel {
		return w.Writer.Write(p)
	}
	return len(p), nil
}

func configureManager(f *flags) ctrl.Manager {
	tlsOpts := configureTLSOptions(f)
	metricsCertWatcher, metricsServerOptions := setupMetricsTLS(f, tlsOpts)
	webhookCertWatcher, webhookTLSOpts := setupWebhookTLS(f, tlsOpts)
	mgr := createManager(f, metricsServerOptions, webhookTLSOpts)
	addCertWatchers(mgr, metricsCertWatcher, webhookCertWatcher)
	setupPluginGenerator(f, mgr)
	setupControllers(mgr)
	setupWebhooks(mgr)
	setupHealthChecks(mgr)

	return mgr
}

func setupPluginGenerator(f *flags, m ctrl.Manager) {
	if f.argocdPluginGenAddr != "0" {
		pluginGenerator, err := argocdplugingenerator.New(m.GetClient(), m.GetCache(), f.argocdPluginGenAddr)
		if err != nil {
			log.Fatal().Msgf("failed to create plugin generator: %v", err)
		}
		err = m.Add(pluginGenerator)
		if err != nil {
			log.Fatal().Msgf("failed to add plugin generator: %v", err)
		}
		err = m.AddReadyzCheck("argocd plugin generator", pluginGenerator.StartedChecker())
		if err != nil {
			log.Fatal().Msgf("failed to add ArgoCD plugin generator readiness check: %v", err)
		}
		err = m.AddHealthzCheck("argocd plugin generator", pluginGenerator.StartedChecker())
		if err != nil {
			log.Fatal().Msgf("failed to add ArgoCD plugin generator health check: %v", err)
		}
	}
}

func configureTLSOptions(f *flags) []func(*tls.Config) {
	var tlsOpts []func(*tls.Config)
	if !f.enableHTTP2 {
		tlsOpts = append(tlsOpts, func(c *tls.Config) {
			log.Info().Msg("disabling http/2")
			c.NextProtos = []string{"http/1.1"}
		})
	}
	return tlsOpts
}

func setupWebhookTLS(f *flags, tlsOpts []func(*tls.Config)) (*certwatcher.CertWatcher, []func(*tls.Config)) {
	if len(f.webhookCertPath) == 0 {
		return nil, tlsOpts
	}

	log.Info().Msgf("initializing webhook certificate watcher using provided certificates: "+
		"webhook-cert-path=%s, webhook-cert-name=%s, webhook-cert-key=%s",
		f.webhookCertPath, f.webhookCertName, f.webhookCertKey)

	watcher, err := certwatcher.New(
		filepath.Join(f.webhookCertPath, f.webhookCertName),
		filepath.Join(f.webhookCertPath, f.webhookCertKey),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize webhook certificate watcher")
	}

	tlsOpts = append(tlsOpts, func(config *tls.Config) {
		config.GetCertificate = watcher.GetCertificate
	})

	return watcher, tlsOpts
}

func setupMetricsTLS(f *flags, tlsOpts []func(*tls.Config)) (*certwatcher.CertWatcher, metricsserver.Options) {
	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	options := metricsserver.Options{
		BindAddress:   f.metricsAddr,
		SecureServing: f.secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if f.secureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint.
		options.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	// If the certificate is not specified, controller-runtime will automatically
	// generate self-signed certificates for the metrics server. While convenient for development and testing,
	// this setup is not recommended for production.
	if len(f.metricsCertPath) == 0 {
		return nil, options
	}

	log.Info().Msgf("initializing metrics certificate watcher using provided certificates: "+
		"metrics-cert-path=%s, metrics-cert-name=%s, metrics-cert-key=%s",
		f.metricsCertPath, f.metricsCertName, f.metricsCertKey)

	watcher, err := certwatcher.New(
		filepath.Join(f.metricsCertPath, f.metricsCertName),
		filepath.Join(f.metricsCertPath, f.metricsCertKey),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize metrics certificate watcher")
	}

	options.TLSOpts = append(options.TLSOpts, func(config *tls.Config) {
		config.GetCertificate = watcher.GetCertificate
	})

	return watcher, options
}

func createManager(f *flags, metricsServerOptions metricsserver.Options,
	webhookTLSOpts []func(*tls.Config),
) ctrl.Manager {
	webhookServer := webhook.NewServer(webhook.Options{TLSOpts: webhookTLSOpts})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: f.probeAddr,
		LeaderElection:         f.enableLeaderElection,
		LeaderElectionID:       "74669540.cpet.belastingdienst.nl",
	})
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create manager")
	}
	return mgr
}

func addCertWatchers(mgr ctrl.Manager, metricsWatcher, webhookWatcher *certwatcher.CertWatcher) {
	if metricsWatcher != nil {
		log.Info().Msg("adding metrics certificate watcher to manager")
		if err := mgr.Add(metricsWatcher); err != nil {
			log.Fatal().Err(err).Msg("unable to add metrics certificate watcher to manager")
		}
	}
	if webhookWatcher != nil {
		log.Info().Msg("adding webhook certificate watcher to manager")
		if err := mgr.Add(webhookWatcher); err != nil {
			log.Fatal().Err(err).Msg("unable to add webhook certificate watcher to manager")
		}
	}
}

func setupControllers(mgr ctrl.Manager) {
	if err := config.SetupPaasConfigInformer(mgr); err != nil {
		log.Fatal().Err(err).Msg("unable to set up PaasConfig informer")
	}

	if err := (&controller.PaasConfigReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Fatal().Err(err).Str("controller", "PaasConfig").Msg("unable to create controller")
	}

	if err := (&controller.PaasReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Fatal().Err(err).Str("controller", "Paas").Msg("unable to create controller")
	}
}

func setupHealthChecks(mgr ctrl.Manager) {
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Fatal().Err(err).Msg("unable to set up health check")
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Fatal().Err(err).Msg("unable to set up ready check")
	}
}

func setupWebhooks(mgr ctrl.Manager) {
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err := webhookv1alpha1.SetupPaasWebhookWithManager(mgr); err != nil {
			log.Fatal().Err(err).Str("webhook", "Paas").Msg(webhookErrMsg)
		}
		if err := webhookv1alpha2.SetupPaasWebhookWithManager(mgr); err != nil {
			log.Fatal().Err(err).Str("webhook", "Paas").Msg(webhookErrMsg)
		}
		if err := webhookv1alpha1.SetupPaasConfigWebhookWithManager(mgr); err != nil {
			log.Fatal().Err(err).Str("webhook", "PaasConfig").Msg(webhookErrMsg)
		}
		if err := webhookv1alpha2.SetupPaasConfigWebhookWithManager(mgr); err != nil {
			log.Fatal().Err(err).Str("webhook", "PaasConfig").Msg(webhookErrMsg)
		}
		if err := webhookv1alpha1.SetupPaasNsWebhookWithManager(mgr); err != nil {
			log.Fatal().Err(err).Str("webhook", "PaasNS").Msg(webhookErrMsg)
		}
		if err := webhookv1alpha2.SetupPaasNsWebhookWithManager(mgr); err != nil {
			log.Fatal().Err(err).Str("webhook", "PaasNS").Msg(webhookErrMsg)
		}
	}
}
