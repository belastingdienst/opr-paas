/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/config"
	"github.com/belastingdienst/opr-paas/internal/controller"
	"github.com/belastingdienst/opr-paas/internal/logging"
	gitopsresources "github.com/belastingdienst/opr-paas/internal/stubs/argoproj-labs/v1beta1"
	argoresources "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/version"
	webhookv1alpha1 "github.com/belastingdienst/opr-paas/internal/webhook/v1alpha1"
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
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	// +kubebuilder:scaffold:imports
)

var scheme = runtime.NewScheme()

type flags struct {
	metricsAddr          string
	enableLeaderElection bool
	probeAddr            string
	getVersion           bool
	pretty               bool
	debug                bool
	componentDebugList   string
	splitLogOutput       bool
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(quotav1.AddToScheme(scheme))
	utilruntime.Must(userv1.AddToScheme(scheme))
	utilruntime.Must(argoresources.AddToScheme(scheme))
	utilruntime.Must(gitopsresources.AddToScheme(scheme))

	utilruntime.Must(v1alpha1.AddToScheme(scheme))
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
		log.Fatal().Err(err).Msg("problem running manager")
	}
}

func configureFlags() *flags {
	f := &flags{}
	flag.StringVar(&f.metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&f.probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&f.getVersion, "version", false, "Print version and quit")
	flag.BoolVar(&f.enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
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

	if debug {
		if componentDebugList != "" {
			log.Fatal().Msg("cannot pass --debug and --component-debug simultaneously")
		}
	} else if componentDebugList != "" {
		logging.SetComponentDebug(strings.Split(componentDebugList, ","))
		log.Logger = log.Level(zerolog.InfoLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	ctrl.SetLogger(zerologr.New(&log.Logger))
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
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                server.Options{BindAddress: f.metricsAddr},
		HealthProbeBindAddress: f.probeAddr,
		LeaderElection:         f.enableLeaderElection,
		LeaderElectionID:       "74669540.cpet.belastingdienst.nl",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("unable to start manager")
	}

	go func() {
		config.Watch(mgr.GetConfig(), mgr.GetHTTPClient(), mgr.GetScheme())
	}()

	if err = (&controller.PaasConfigReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Fatal().Err(err).Str("controller", "PaasConfig").Msg("unable to create controller")
	}
	if err = (&controller.PaasReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Fatal().Err(err).Str("controller", "Paas").Msg("unable to create controller")
	}
	if err = (&controller.PaasNSReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Fatal().Err(err).Str("controller", "PaasNS").Msg("unable to create controller")
	}
	// +kubebuilder:scaffold:builder

	configureWebhooks(mgr)
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Fatal().Err(err).Msg("unable to set up health check")
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Fatal().Err(err).Msg("unable to set up ready check")
	}

	return mgr
}

func configureWebhooks(mgr ctrl.Manager) {
	// nolint:goconst
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err := webhookv1alpha1.SetupPaasWebhookWithManager(mgr); err != nil {
			log.Fatal().Err(err).Str("webhook", "Paas").Msg("unable to create webhook")
		}
		if err := webhookv1alpha1.SetupPaasConfigWebhookWithManager(mgr); err != nil {
			log.Fatal().Err(err).Str("webhook", "PaasConfig").Msg("unable to create webhook")
		}
		if err := webhookv1alpha1.SetupPaasNsWebhookWithManager(mgr); err != nil {
			log.Fatal().Err(err).Str("webhook", "PaasNS").Msg("unable to create webhook")
		}
	}
}
