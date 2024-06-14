package dynamic

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	applisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/klog/v2"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"

	"github.com/gocrane/crane-scheduler/pkg/controller/annotator"
	"github.com/gocrane/crane-scheduler/pkg/plugins/apis/config"
	"github.com/gocrane/crane-scheduler/pkg/plugins/apis/policy"
	"github.com/gocrane/crane-scheduler/pkg/plugins/metrics"
	"github.com/gocrane/crane-scheduler/pkg/utils"
)

var _ framework.FilterPlugin = &DynamicScheduler{}
var _ framework.ScorePlugin = &DynamicScheduler{}
var _ framework.PreScorePlugin = &DynamicScheduler{}
var _ framework.PostBindPlugin = &DynamicScheduler{}

const (
	// Name is the name of the plugin used in the plugin registry and configurations.
	Name = "Dynamic"
)

// Dynamic-scheduler is a real load-aware scheduler plugin.
type DynamicScheduler struct {
	handle          framework.FrameworkHandle
	schedulerPolicy *policy.DynamicSchedulerPolicy
	bindingRecords  *annotator.BindingRecords
	deployLister    applisters.DeploymentLister
}

// Name returns name of the plugin.
func (ds *DynamicScheduler) Name() string {
	return Name
}

// Filter invoked at the filter extension point.
// checkes if the real load of one node is too high.
// It returns a list of failure reasons if the node is overload.
func (ds *DynamicScheduler) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	// ignore daemonset pod
	if utils.IsDaemonsetPod(pod) {
		return framework.NewStatus(framework.Success, "")
	}

	node := nodeInfo.Node()
	if node == nil {
		return framework.NewStatus(framework.Error, "node not found")
	}

	nodeAnnotations, nodeName := nodeInfo.Node().Annotations, nodeInfo.Node().Name
	if nodeAnnotations == nil {
		nodeAnnotations = map[string]string{}
	}

	for _, policy := range ds.schedulerPolicy.Spec.Predicate {
		// Ignore applicationResourceAware plugin policy
		if strings.HasPrefix(policy.Name, utils.RangePrefix) {
			continue
		}

		activeDuration, err := utils.GetActiveDuration(ds.schedulerPolicy.Spec.SyncPeriod, policy.Name)
		if err != nil || activeDuration == 0 {
			klog.V(5).Infof("Plugin[%s] failed to get active duration: %v", ds.Name(), err)
			continue
		}

		if isOverLoad(nodeName, nodeAnnotations, policy, activeDuration) {
			return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("Load[%s] of node[%s] is too high", policy.Name, nodeName))
		}

	}
	return framework.NewStatus(framework.Success, "")
}

// PreScore invoked at the PreScore extension point.
// It records the number of available nodes for priority to metrics && logs.
func (ds *DynamicScheduler) PreScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodes []*v1.Node) *framework.Status {
	klog.V(5).Infof("Plugin[%s] pod %s/%s available scoring nodes number %d", ds.Name(), pod.Namespace, pod.Name, len(nodes))

	defer func() {
		metrics.DynamicPriorityAvailableNodesNumber.WithLabelValues("PreScore").Set(float64(len(nodes)))
	}()

	return framework.NewStatus(framework.Success, "")
}

// Score invoked at the Score extension point.
// It gets metric data from node annotation, and favors nodes with the least real resource usage.
func (ds *DynamicScheduler) Score(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) (int64, *framework.Status) {
	nodeInfo, err := ds.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("getting node %q from Snapshot: %v", nodeName, err))
	}

	node := nodeInfo.Node()
	if node == nil {
		return 0, framework.NewStatus(framework.Error, "node not found")
	}

	nodeAnnotations := node.Annotations
	if nodeAnnotations == nil {
		nodeAnnotations = map[string]string{}
	}

	// 计算节点热点值
	var hotValue float32
	for _, hv := range ds.schedulerPolicy.Spec.HotValue {
		hotValue += float32(ds.bindingRecords.GetLastNodeBindingCount(nodeName, hv.TimeRange.Duration)) / float32((hv.Count))
	}

	score := getNodeScore(node.Name, nodeAnnotations, ds.schedulerPolicy.Spec)
	finalScore := utils.NormalizeScore(int64(score-int(hotValue*10)), framework.MaxNodeScore, framework.MinNodeScore)
	klog.V(5).Infof("Plugin[%s] Node[%s]'s final score is %d, while score is %d and hotValue is %.2f", ds.Name(), node.Name, finalScore, score, hotValue)

	return finalScore, nil
}

func (ds *DynamicScheduler) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

// pod调度完成以后将调度信息记录到bindingRecords中，用于后续的热点值计算
func (ds *DynamicScheduler) PostBind(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) {
	binding := &annotator.Binding{
		Node:      nodeName,
		Namespace: p.Namespace,
		PodName:   p.Name,
		Timestamp: time.Now().UTC().Unix(),
	}
	ds.bindingRecords.AddBinding(binding)
	klog.V(4).Infof("[crane - %s] pod %s/%s post bind to node %s", ds.Name(), p.Namespace, p.Name, nodeName)
}

// NewDynamicScheduler returns a Crane Scheduler object.
func NewDynamicScheduler(plArgs runtime.Object, h framework.FrameworkHandle) (framework.Plugin, error) {
	args, ok := plArgs.(*config.DynamicArgs)
	if !ok {
		return nil, fmt.Errorf("want args to be of type DynamicArgs, got %T.", plArgs)
	}

	schedulerPolicy, err := utils.LoadPolicyFromFile(args.PolicyConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduler policy from config file: %v", err)
	}

	// 注册metrics
	metrics.RegisterDynamicSchedulerMetrics()

	dynamicScheduler := &DynamicScheduler{
		schedulerPolicy: schedulerPolicy,
		handle:          h,
		bindingRecords:  annotator.NewBindingRecords(1024, 5*time.Minute),
		deployLister:    h.SharedInformerFactory().Apps().V1().Deployments().Lister(),
	}

	// 定时清理bindingRecords heap中过期的数据
	stopCh := make(chan struct{})
	go wait.Until(dynamicScheduler.bindingRecords.BindingsGC, time.Minute, stopCh)

	return dynamicScheduler, nil
}
