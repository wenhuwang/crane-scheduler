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
		if strings.HasPrefix(policy.Name, RangePrefix) {
			nameSlice, err := parsePolicyName(policy.Name)
			if err != nil {
				continue
			}
			resource, deployMetricName, nodeMetricName := nameSlice[1], nameSlice[2], nameSlice[3]
			activeDuration, err := dynamic.GetActiveDuration(ara.schedulerPolicy.Spec.SyncPeriod, nodeMetricName)
			if err != nil || activeDuration == 0 {
				klog.Warningf("[crane-%s] failed to get active duration: %v", ara.Name(), err)
				continue
			}
			nodeUsages, err := getResourceUsage(node.Annotations, nodeMetricName, activeDuration)
			if err != nil {
				klog.Warningf("[crane-%s] can not get the usage of resource[%s] from node[%s]'s annotation: %v", ara.Name(), policy.Name, node.Name, err)
				continue
			}

			activeDuration, err = dynamic.GetActiveDuration(ara.schedulerPolicy.Spec.SyncAppPeriod, deployMetricName)
			if err != nil || activeDuration == 0 {
				klog.Warningf("[crane-%s] failed to get active duration: %v", ara.Name(), err)
				continue
			}
			deployUsages, err := getResourceUsage(deploy.Annotations, deployMetricName, activeDuration)
			if err != nil {
				klog.Warningf("[crane-%s] can not get the usage of resource[%s] from deployment[%s]'s annotation: %v", ara.Name(), policy.Name, node.Name, err)
				continue
			}

			var nodeAllocation int64
			if resource == "cpu" {
				nodeAllocation = nodeInfo.Allocatable.MilliCPU
				klog.V(4).Infof("[crane-%s] node[%s] cpu allocation is %d", ara.Name(), node.Name, nodeAllocation)
			} else if resource == "memory" {
				nodeAllocation = nodeInfo.Allocatable.Memory
				klog.V(4).Infof("[crane-%s] node[%s] memory allocation is %d", ara.Name(), node.Name, nodeAllocation)
			}
			if predictingOverLoad(nodeUsages, deployUsages, policy, nodeAllocation) {
				return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("Plugin[%s] policy[%s] of node[%s] is too high", ara.Name(), policy.Name, node.Name))
			}
		}
	}

	return nil
}
