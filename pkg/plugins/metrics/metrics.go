package metrics

import (
	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
)

const DynamicSchedulerSubsystem = "scheduler_dynamic"

var (
	DynamicPriorityAvailableNodesNumber = metrics.NewGaugeVec(
		&metrics.GaugeOpts{
			Subsystem: DynamicSchedulerSubsystem,
			Name:      "priority_available_nodes_number",
			Help:      "Number of available nodes for priority",
		},
		[]string{"extension_point"},
	)
)

// RegisterDynamicSchedulerMetrics is used for register dynamic scheduler plugin metrics.
func RegisterDynamicSchedulerMetrics() {
	legacyregistry.MustRegister(DynamicPriorityAvailableNodesNumber)
}

const ApplicationResourceAwareSubsystem = "scheduler_ara"

var (
	ApplicationResourceAwareFilterFiltedNodes = metrics.NewCounterVec(
		&metrics.CounterOpts{
			Subsystem: ApplicationResourceAwareSubsystem,
			Name:      "filter_filted_nodes_total",
			Help:      "Number of filted nodes for filter",
		},
		[]string{"extension_point", "node"},
	)
)

// RegisterAraSchedulerMetrics is used for register ApplicationResourceAware scheduler plugin metrics.
func RegisterAraSchedulerMetrics() {
	legacyregistry.MustRegister(ApplicationResourceAwareFilterFiltedNodes)
}
