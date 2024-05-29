package applicationresourceaware

import (
	"fmt"
	"strings"
	"time"

	"github.com/gocrane/crane-scheduler/pkg/plugins/apis/policy"
	"github.com/gocrane/crane-scheduler/pkg/plugins/dynamic"
	"github.com/gocrane/crane-scheduler/pkg/utils"
	"k8s.io/klog/v2"
)

const (
	// ExtraActivePeriod gives extra active time to the annotation.
	ExtraActivePeriod = 5 * time.Minute
)

func predictingOverLoad(nodeUsageStr, deployUsageStr string, policy policy.PredicatePolicy, nodeCapacity int64, nodeName string) bool {
	// threshold was set as 0 means that the filter according to this metric is useless.
	if policy.MaxLimitPecent == 0 {
		klog.V(4).Infof("[%s] ignore the filter of resource[%s] for MaxLimitPecent was set as 0", Name, policy.Name)
		return false
	}

	nodeUsage, err := utils.ParseRangeMetricsByString(nodeUsageStr)
	if err != nil {
		return false
	}
	deployUsage, err := utils.ParseRangeMetricsByString(deployUsageStr)
	if err != nil {
		return false
	}
	if len(nodeUsage) != len(deployUsage) {
		klog.V(4).Info("node or deployment usage annotations values illegel")
		return false
	}

	usages := make([]float64, len(nodeUsage))
	for i, nu := range nodeUsage {
		usage := (nu*float64(nodeCapacity) + deployUsage[i]) / float64(nodeCapacity)
		if usage > policy.MaxLimitPecent {
			return true
		}
		usages[i] = usage
	}
	klog.V(6).Infof("[%s] node[%s] policy[%s] predicted usages is %v", Name, nodeName, policy.Name, usages)
	return false
}

func parsePolicyName(name string) ([]string, error) {
	nameSlice := strings.Split(name, ":")
	if len(nameSlice) != 4 {
		return nil, fmt.Errorf("illegel value: %s", name)
	}

	return nameSlice, nil
}

func getResourceUsage(anno map[string]string, key string, activeDuration time.Duration) (string, error) {
	usedstr, ok := anno[key]
	if !ok {
		return "", fmt.Errorf("key[%s] not found", usedstr)
	}

	usedSlice := strings.Split(usedstr, ",")
	if len(usedSlice) != 2 {
		return "", fmt.Errorf("illegel value: %s", usedstr)
	}

	if !dynamic.InActivePeriod(usedSlice[1], activeDuration) {
		return "", fmt.Errorf("timestamp[%s] is expired", usedstr)
	}

	return usedSlice[0], nil
}
