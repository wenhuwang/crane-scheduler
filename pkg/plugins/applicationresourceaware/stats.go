package applicationresourceaware

import (
	"fmt"
	"strings"
	"time"

	"github.com/gocrane/crane-scheduler/pkg/plugins/apis/policy"
	"github.com/gocrane/crane-scheduler/pkg/utils"
	"k8s.io/klog/v2"
)

const (
	// ExtraActivePeriod gives extra active time to the annotation.
	ExtraActivePeriod = 5 * time.Minute
)

func predictingOverLoad(nodeUsageStr, deployUsageStr, deltaUsageStr string, policy policy.PredicatePolicy, nodeCapacity int64, nodeName string) bool {
	// threshold was set as 0 means that the filter according to this metric is useless.
	if policy.MaxLimitPecent == 0 {
		klog.V(5).Infof("Plugin[%s] ignore the filter of resource[%s] for MaxLimitPecent was set as 0", Name, policy.Name)
		return false
	}

	nodeUsage, err := utils.ParseRangeMetricsByString(nodeUsageStr)
	if err != nil {
		klog.V(5).Infof("Plugin[%s] parse node %s range metrics failed: %v", Name, nodeName, err)
		return false
	}
	deployUsage, err := utils.ParseRangeMetricsByString(deployUsageStr)
	if err != nil {
		klog.V(5).Infof("Plugin[%s] parse deployment range metrics failed: %v", Name, err)
		return false
	}

	var deltaUsage []float64
	if deltaUsageStr == "" {
		deltaUsage = make([]float64, len(nodeUsage))
	} else {
		deltaUsage, err = utils.ParseRangeMetricsByString(deltaUsageStr)
		if err != nil {
			klog.V(5).Infof("Plugin[%s] parse node %s delta metrics failed: %v", Name, nodeName, err)
			return false
		}
	}

	var deployInvalid, deltaInvalid bool
	if len(nodeUsage) != len(deployUsage) {
		klog.V(5).Infof("Plugin[%s] node %s and deployment usage annotations values not match", Name, nodeName)
		deployInvalid = true
	}

	if len(nodeUsage) != len(deltaUsage) {
		klog.V(5).Infof("Plugin[%s] node %s delta usage annotations values illegel", Name, nodeName)
		deltaInvalid = true
	}

	for i, nu := range nodeUsage {
		usage := nu * float64(nodeCapacity)
		if !deltaInvalid {
			usage += deltaUsage[i]
		}

		if !deployInvalid {
			usage += deployUsage[i]
		}
		utilization := usage / float64(nodeCapacity)
		if utilization > policy.MaxLimitPecent {
			return true
		}
	}
	return false
}

func parsePolicyName(name string) ([]string, error) {
	nameSlice := strings.Split(name, ":")
	if len(nameSlice) != 4 {
		return nil, fmt.Errorf("illegel value: %s", name)
	}

	return nameSlice, nil
}
