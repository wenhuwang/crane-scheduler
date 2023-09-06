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
