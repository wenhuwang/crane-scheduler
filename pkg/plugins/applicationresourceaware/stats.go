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

	RangePrefix = "range"
)

func predictingOverLoad(nodeUsageStr, deployUsageStr string, policy policy.PredicatePolicy, nodeAllocation int64) bool {
	// threshold was set as 0 means that the filter according to this metric is useless.
	if policy.MaxLimitPecent == 0 {
		klog.V(4).Infof("[crane] ignore the filter of resource[%s] for MaxLimitPecent was set as 0", policy.Name)
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

	for i, nu := range nodeUsage {
		if (nu*float64(nodeAllocation)+deployUsage[i])/float64(nodeAllocation) > policy.MaxLimitPecent {
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
