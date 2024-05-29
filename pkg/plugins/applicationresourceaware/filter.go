package applicationresourceaware

import (
	"context"
	"fmt"
	"strings"

	"github.com/gocrane/crane-scheduler/pkg/plugins/dynamic"
	"github.com/gocrane/crane-scheduler/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

func (ara *ApplicationResourceAware) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	// only for deployment pods
	deployName, err := utils.GetDeploymentNameByPod(pod, ara.replicasetLister)
	if err != nil {
		klog.V(6).Infof("get deployment name by pod failed: %v", err)
		return nil
	}
	deploy, err := ara.deploymentLister.Deployments(pod.Namespace).Get(deployName)
	if err != nil {
		klog.V(6).Infof("get deployment by name failed: %v", err)
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
			activeDuration, err := dynamic.GetActiveDuration(ara.schedulerPolicy.Spec.SyncPeriod, nodeMetricName)
			if err != nil || activeDuration == 0 {
				klog.Warningf("[%s] failed to get active duration: %v", ara.Name(), err)
				continue
			}
			nodeUsages, err := getResourceUsage(node.Annotations, nodeMetricName, activeDuration)
			if err != nil {
				klog.Warningf("[%s] can not get the usage of resource[%s] from node[%s]'s annotation: %v", ara.Name(), nodeMetricName, node.Name, err)
				continue
			}

			activeDuration, err = dynamic.GetActiveDuration(ara.schedulerPolicy.Spec.SyncAppPeriod, deployMetricName)
			if err != nil || activeDuration == 0 {
				klog.Warningf("[%s] failed to get active duration: %v", ara.Name(), err)
				continue
			}
			deployUsages, err := getResourceUsage(deploy.Annotations, deployMetricName, activeDuration)
			if err != nil {
				klog.Warningf("[%s] can not get the usage of resource[%s] from deployment[%s]'s annotation: %v", ara.Name(), deployMetricName, deploy.Name, err)
				continue
			}

			var nodeCapacity int64
			if resource == "cpu" {
				nodeCapacity = node.Status.Capacity.Cpu().Value()
				klog.V(4).Infof("[%s] node[%s] cpu capacity is %d", ara.Name(), node.Name, nodeCapacity)
			} else if resource == "memory" {
				nodeCapacity = node.Status.Capacity.Memory().Value()
				klog.V(4).Infof("[%s] node[%s] memory capacity is %d", ara.Name(), node.Name, nodeCapacity)
			}
			if predictingOverLoad(nodeUsages, deployUsages, policy, nodeCapacity, node.Name) {
				return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("Plugin[%s] node[%s] policy[%s] for pod[%s] is too high", ara.Name(), node.Name, policy.Name, pod.Name))
			}
		}
	}

	return nil
}
