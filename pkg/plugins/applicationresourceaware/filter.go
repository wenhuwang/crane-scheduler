package applicationresourceaware

import (
	"context"
	"fmt"
	"strings"

	"github.com/gocrane/crane-scheduler/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

func (ara *ApplicationResourceAware) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	// only for deployment pods
	deploy, err := utils.GetDeploymentByPod(pod, ara.replicasetLister, ara.deploymentLister)
	if err != nil {
		klog.V(6).Infof("get deployment by pod failed: %v", err)
		return nil
	}

	node := nodeInfo.Node()
	if node == nil {
		return framework.NewStatus(framework.Error, "node not found")
	}

	nodeAnnotations := nodeInfo.Node().Annotations
	if nodeAnnotations == nil {
		return nil
	}

	// predicate policy naming rules is range:RESOURCE:APP_METRIC_NAME:NODE_METRIC_NAME
	for _, policy := range ara.schedulerPolicy.Spec.Predicate {
		if strings.HasPrefix(policy.Name, utils.RangePrefix) {
			nameSlice, err := parsePolicyName(policy.Name)
			if err != nil {
				continue
			}
			resource, deployMetricName, nodeMetricName := nameSlice[1], nameSlice[2], nameSlice[3]
			activeDuration, err := utils.GetActiveDuration(ara.schedulerPolicy.Spec.SyncPeriod, nodeMetricName)
			if err != nil || activeDuration == 0 {
				klog.Warningf("Plugin[%s] failed to get active duration: %v", ara.Name(), err)
				continue
			}
			nodeUsages, err := utils.GetResourceUsageRange(node.Annotations, nodeMetricName, activeDuration)
			if err != nil {
				klog.Warningf("Plugin[%s] can not get the usage of resource[%s] from node[%s]'s annotation: %v", ara.Name(), nodeMetricName, node.Name, err)
				continue
			}

			activeDuration, err = utils.GetActiveDuration(ara.schedulerPolicy.Spec.SyncAppPeriod, deployMetricName)
			if err != nil || activeDuration == 0 {
				klog.Warningf("Plugin[%s] failed to get active duration: %v", ara.Name(), err)
				continue
			}
			deployUsages, err := utils.GetResourceUsageRange(deploy.Annotations, deployMetricName, activeDuration)
			if err != nil {
				klog.Warningf("Plugin[%s] can not get the usage of resource[%s] from deployment[%s]'s annotation: %v", ara.Name(), deployMetricName, deploy.Name, err)
				continue
			}

			deltaName := utils.DeltaPrefixName + deployMetricName
			deltaUsages, err := utils.GetDeltaUsageRange(node.Annotations, deltaName)
			if err != nil {
				klog.Warningf("Plugin[%s] get node %s metrics %s from annotation failed: %v", ara.Name(), node.Name, deltaName, err)
				continue
			}

			var nodeCapacity int64
			if resource == "cpu" {
				nodeCapacity = node.Status.Capacity.Cpu().Value()
			} else if resource == "memory" {
				nodeCapacity = node.Status.Capacity.Memory().Value()
			}
			if predictingOverLoad(nodeUsages, deployUsages, deltaUsages, policy, nodeCapacity, node.Name) {
				klog.V(4).Infof("Plugin[%s] node[%s] policy[%s] for pod[%s] is too high", ara.Name(), node.Name, policy.Name, pod.Name)
				return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("Plugin[%s] node[%s] policy[%s] for pod[%s] is too high", ara.Name(), node.Name, policy.Name, pod.Name))
			}
		}
	}

	return nil
}
