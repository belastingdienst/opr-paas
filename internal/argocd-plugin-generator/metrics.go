/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package argocd_plugin_generator

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// PluginGeneratorRequestTotal is a prometheus metric which is a counter of
	// the total processed plugin generator requests.
	PluginGeneratorRequestTotal = func() *prometheus.CounterVec {
		return prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "opr_paas_plugin_generator_requests_total",
				Help: "Total number of plugin generator requests by HTTP status code.",
			},
			[]string{"code"},
		)
	}()
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(PluginGeneratorRequestTotal)
}
