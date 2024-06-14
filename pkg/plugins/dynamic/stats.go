package dynamic

import (
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	"github.com/gocrane/crane-scheduler/pkg/plugins/apis/policy"
	"github.com/gocrane/crane-scheduler/pkg/utils"
)

const (
	// NodeHotValue is the key of hot value annotation.
	NodeHotValue = "node_hot_value"
	// DefautlHotVauleActivePeriod defines the validity period of nodes' hotvalue.
	DefautlHotVauleActivePeriod = 5 * time.Minute
)

func getScore(anno map[string]string, priorityPolicy policy.PriorityPolicy, syncPeriod []policy.SyncPolicy) (float64, error) {
	activeDuration, err := utils.GetActiveDuration(syncPeriod, priorityPolicy.Name)
	if err != nil || activeDuration == 0 {
		return 0, fmt.Errorf("failed to get the active duration of resource[%s]: %v, while the actual value is %v", priorityPolicy.Name, err, activeDuration)
	}

	usage, err := utils.GetResourceUsage(anno, priorityPolicy.Name, activeDuration)
	if err != nil {
		return 0, err
	}

	score := (1. - usage) * priorityPolicy.Weight * float64(framework.MaxNodeScore)

	return score, nil
}

func isOverLoad(name string, anno map[string]string, predicatePolicy policy.PredicatePolicy, activeDuration time.Duration) bool {
	usage, err := utils.GetResourceUsage(anno, predicatePolicy.Name, activeDuration)
	if err != nil {
		klog.V(5).Infof("[crane] can not get the usage of resource[%s] from node[%s]'s annotation: %v", predicatePolicy.Name, name, err)
		return false
	}

	// threshold was set as 0 means that the filter according to this metric is useless.
	if predicatePolicy.MaxLimitPecent == 0 {
		klog.V(6).Info("[crane] ignore the filter of resource[%s] for MaxLimitPecent was set as 0")
		return false
	}

	if usage > predicatePolicy.MaxLimitPecent {
		return true
	}

	return false
}

func getNodeScore(name string, anno map[string]string, policySpec policy.PolicySpec) int {

	lenPriorityPolicyList := len(policySpec.Priority)
	if lenPriorityPolicyList == 0 {
		klog.V(6).Infof("[crane] no priority policy exists, all nodes scores 0.")
		return 0
	}

	var score, weight float64

	for _, priorityPolicy := range policySpec.Priority {

		priorityScore, err := getScore(anno, priorityPolicy, policySpec.SyncPeriod)
		if err != nil {
			klog.V(5).Infof("[crane] failed to get node 's score: %v", name, priorityPolicy.Name, score)
		}

		weight += priorityPolicy.Weight
		score += priorityScore
	}

	finnalScore := int(score / weight)

	return finnalScore
}

func getNodeHotValue(node *v1.Node) float64 {
	anno := node.ObjectMeta.Annotations
	if anno == nil {
		return 0
	}

	hotvalue, err := utils.GetResourceUsage(anno, NodeHotValue, DefautlHotVauleActivePeriod)
	if err != nil {
		return 0
	}

	klog.V(6).Infof("[crane] Node[%s]'s hotvalue is %f\n", node.Name, hotvalue)

	return hotvalue
}
