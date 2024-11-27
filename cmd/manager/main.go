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

	gitopsResources "github.com/belastingdienst/opr-paas/internal/stubs/argoproj-labs/v1beta1"
	argoResources "github.com/belastingdienst/opr-paas/internal/stubs/argoproj/v1alpha1"
	quotav1 "github.com/openshift/api/quota/v1"
	userv1 "github.com/openshift/api/user/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/controller"
	"github.com/belastingdienst/opr-paas/internal/version"
	//+kubebuilder:scaffold:imports
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(quotav1.AddToScheme(scheme))
	utilruntime.Must(userv1.AddToScheme(scheme))
	utilruntime.Must(argoResources.AddToScheme(scheme))
	utilruntime.Must(gitopsResources.AddToScheme(scheme))

	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var getVersion bool
	var pretty bool
	var debug bool
	var componentDebugList string
	var splitLogOutput bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&getVersion, "version", false, "Print version and quit")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&pretty, "pretty", false, "Pretty-print logging output")
	flag.BoolVar(&debug, "debug", false, "Log all debug messages")
	flag.StringVar(&componentDebugList, "component-debug", "", "Comma-separated list of components to log debug messages for.")
	flag.BoolVar(&splitLogOutput, "split-log-output", false, "Send error logs to stderr, and the rest to stdout.")

	flag.Parse()
	configureLogging(pretty, debug, componentDebugList, splitLogOutput)

	if getVersion {
		fmt.Printf("opr-paas version %s", version.PaasVersion)
		os.Exit(0)
	} else {
		log.Info().Str("version", version.PaasVersion).Msg("opr-paas version")
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                server.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
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

	if err = (&controller.PaasConfigReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("PaasConfig"),
	}).SetupWithManager(mgr); err != nil {
		os.Exit(1)
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
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Fatal().Err(err).Msg("unable to set up health check")
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Fatal().Err(err).Msg("unable to set up ready check")
	}

	log.Info().Msgf("starting manager version %s", version.PaasVersion)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Fatal().Err(err).Msg("problem running manager")
	}
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
		controller.SetComponentDebug(strings.Split(componentDebugList, ","))
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
