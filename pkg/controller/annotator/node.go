package annotator

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	policy "github.com/gocrane/crane-scheduler/pkg/plugins/apis/policy"

	"github.com/gocrane/crane-scheduler/pkg/controller/prometheus"
	prom "github.com/gocrane/crane-scheduler/pkg/controller/prometheus"
	utils "github.com/gocrane/crane-scheduler/pkg/utils"
	"github.com/robfig/cron/v3"
)

const (
	HotValueKey    = "node_hot_value"
	DefaultBackOff = 10 * time.Second
	MaxBackOff     = 360 * time.Second
)

type nodeController struct {
	*Controller
	queue workqueue.RateLimitingInterface
}

func newNodeController(c *Controller) *nodeController {
	nodeRateLimiter := workqueue.NewItemExponentialFailureRateLimiter(DefaultBackOff,
		MaxBackOff)

	return &nodeController{
		Controller: c,
		queue:      workqueue.NewNamedRateLimitingQueue(nodeRateLimiter, "node_event_queue"),
	}
}

func (n *nodeController) Run() {
	defer n.queue.ShutDown()
	klog.Infof("Start to reconcile node events")

	for n.processNextWorkItem() {
	}
}

func (n *nodeController) processNextWorkItem() bool {
	key, quit := n.queue.Get()
	if quit {
		return false
	}
	defer n.queue.Done(key)

	forget, err := n.syncNode(key.(string))
	if err != nil {
		klog.Warningf("failed to sync this node [%q]: %v", key.(string), err)
	}
	if forget {
		n.queue.Forget(key)
		return true
	}

	n.queue.AddRateLimited(key)
	return true
}

func (n *nodeController) syncNode(key string) (bool, error) {
	startTime := time.Now()
	klog.V(5).Infof("Started syncing node event %q (%v)", key, time.Since(startTime))
	defer func() {
		klog.V(5).Infof("Finished syncing node event %q (%v)", key, time.Since(startTime))
	}()

	nodeName, metricName, err := splitMetaKeyWithMetricName(key)
	if err != nil {
		return true, fmt.Errorf("invalid resource key: %s", key)
	}

	node, err := n.nodeLister.Get(nodeName)
	if err != nil {
		return true, fmt.Errorf("can not find node[%s]: %v", node, err)
	}

	err = annotateNodeLoad(n.promClient, n.kubeClient, node, metricName)
	if err != nil {
		return false, fmt.Errorf("can not annotate node[%s]: %v", node.Name, err)
	}

	err = annotateNodeHotValue(n.kubeClient, n.bindingRecords, node, n.policy)
	if err != nil {
		return false, err
	}

	return true, nil
}

func annotateNodeLoad(promClient prom.PromClient, kubeClient clientset.Interface, node *v1.Node, key string) (err error) {
	if strings.HasPrefix(key, prometheus.PrefixRange) {
		metricName := strings.TrimPrefix(key, prometheus.PrefixRange)
		value, err := promClient.QueryRangeByNodeIP(metricName, getNodeInternalIP(node))
		if err == nil {
			return utils.PatchNodeAnnotation(kubeClient, node, key, value)
		}
		value, err = promClient.QueryRangeByNodeName(metricName, getNodeName(node))
		if err == nil {
			return utils.PatchNodeAnnotation(kubeClient, node, key, value)
		}
	} else {
		value, err := promClient.QueryByNodeIP(key, getNodeInternalIP(node))
		if err == nil {
			return utils.PatchNodeAnnotation(kubeClient, node, key, value)
		}
		value, err = promClient.QueryByNodeName(key, getNodeName(node))
		if err == nil {
			return utils.PatchNodeAnnotation(kubeClient, node, key, value)
		}
	}
	return fmt.Errorf("failed to get data %s{nodeName=%s}: %v", key, node.Name, err)
}

func annotateNodeHotValue(kubeClient clientset.Interface, br *BindingRecords, node *v1.Node, policy policy.DynamicSchedulerPolicy) error {
	var value int

	for _, p := range policy.Spec.HotValue {
		value += br.GetLastNodeBindingCount(node.Name, p.TimeRange.Duration) / p.Count
	}

	return utils.PatchNodeAnnotation(kubeClient, node, HotValueKey, strconv.Itoa(value))
}

func (n *nodeController) CreateMetricSyncTicker(stopCh <-chan struct{}) {
	klog.V(4).Infof("start sync node metrics ticker")
	for _, p := range n.policy.Spec.SyncPeriod {
		enqueueFunc := func(policy policy.SyncPolicy) {
			nodes, err := n.nodeLister.List(labels.Everything())
			if err != nil {
				klog.Errorf("failed to list nodes: %v", err)
				return
			}

			for _, node := range nodes {
				n.queue.Add(handlingMetaKeyWithMetricName(node.Name, policy.Name))
			}
		}

		enqueueFunc(p)

		go func(policy policy.SyncPolicy) {
			ticker := time.NewTicker(policy.Period.Duration)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					enqueueFunc(policy)
				case <-stopCh:
					return
				}
			}
		}(p)
	}
}

func getNodeInternalIP(node *v1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			return addr.Address
		}
	}

	return node.Name
}

func getNodeName(node *v1.Node) string {
	return node.Name
}

type nodeDeltaResetController struct {
	*Controller
	queue workqueue.RateLimitingInterface
}

func newNodeDeltaResetController(c *Controller) *nodeDeltaResetController {
	nodeRateLimiter := workqueue.NewItemExponentialFailureRateLimiter(DefaultBackOff,
		MaxBackOff)

	return &nodeDeltaResetController{
		Controller: c,
		queue:      workqueue.NewNamedRateLimitingQueue(nodeRateLimiter, "node_delta_event_queue"),
	}
}

func (ndr *nodeDeltaResetController) Run() {
	defer ndr.queue.ShutDown()
	klog.Infof("Start to reconcile node events")

	for ndr.processNextWorkItem() {
	}
}

func (ndr *nodeDeltaResetController) processNextWorkItem() bool {
	key, quit := ndr.queue.Get()
	if quit {
		return false
	}
	defer ndr.queue.Done(key)

	forget, err := ndr.syncNode(key.(string))
	if err != nil {
		klog.Warningf("failed to sync this node [%q]: %v", key.(string), err)
	}
	if forget {
		ndr.queue.Forget(key)
		return true
	}

	ndr.queue.AddRateLimited(key)
	return true
}

func (ndr *nodeDeltaResetController) syncNode(key string) (bool, error) {
	startTime := time.Now()
	klog.V(5).Infof("Started syncing node event %q (%v)", key, time.Since(startTime))
	defer func() {
		klog.V(5).Infof("Finished syncing node event %q (%v)", key, time.Since(startTime))
	}()

	node, err := ndr.nodeLister.Get(key)
	if err != nil {
		return true, fmt.Errorf("can not find node[%s]: %v", node, err)
	}

	for k := range node.GetAnnotations() {
		if strings.HasPrefix(k, utils.DeltaPrefixName) {
			if err := utils.DeleteNodeAnnotation(ndr.kubeClient, node, k); err != nil {
				return false, fmt.Errorf("can't delete deleta annotate node[%s]: %v", node.Name, err)
			}
		}
	}

	return true, nil
}

func (ndr *nodeDeltaResetController) CreateMetricResetTicker(stopCh <-chan struct{}) {
	klog.V(4).Infof("start reset node delta annotations ticker")
	c := cron.New()
	c.AddFunc("0 0 * * *", func() {
		nodes, err := ndr.nodeLister.List(labels.Everything())
		if err != nil {
			klog.Errorf("failed to list nodes: %v", err)
			return
		}

		for _, node := range nodes {
			ndr.queue.Add(node.Name)
		}
	})

	c.Start()
}
