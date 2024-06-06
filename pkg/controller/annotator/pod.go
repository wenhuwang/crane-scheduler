package annotator

import (
	"time"

	"github.com/gocrane/crane-scheduler/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type podController struct {
	*Controller
	queue workqueue.RateLimitingInterface
}

func newPodController(c *Controller) *podController {
	podRateLimiter := workqueue.DefaultControllerRateLimiter()

	return &podController{
		Controller: c,
		queue:      workqueue.NewNamedRateLimitingQueue(podRateLimiter, "pod_event_queue"),
	}
}

func (p *podController) Run() {
	defer p.queue.ShutDown()
	klog.Infof("Start to reconcile Pod events")

	for p.processNextWorkItem() {
	}
}

func (p *podController) processNextWorkItem() bool {
	key, quit := p.queue.Get()
	if quit {
		return false
	}
	defer p.queue.Done(key)

	err := p.reconcile(key.(string))
	if err != nil {
		klog.Warningf("failed to sync this Pod[%q]: %v", key.(string), err)
	}

	return true
}

func (p *podController) handles() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		// AddFunc:    p.handleAddEvent,
		UpdateFunc: p.handleUpdateEvent,
	}
}

// func (p *podController) handleAddEvent(obj interface{}) {
// 	pod := obj.(*corev1.Pod)
// 	for _, condition := range pod.Status.Conditions {
// 		if condition.Type == corev1.PodScheduled && condition.Status == corev1.ConditionTrue {
// 			p.enqueue(obj, cache.Added)
// 			return
// 		}
// 	}
// }

func (p *podController) handleUpdateEvent(old, new interface{}) {
	oldPod, newPod := old.(*corev1.Pod), new.(*corev1.Pod)
	// 只关注pod调度事件和删除事件
	if oldPod.Spec.NodeName != newPod.Spec.NodeName {
		p.enqueue(new, "Scheduled")
	} else if oldPod.DeletionTimestamp == nil && newPod.DeletionTimestamp != nil {
		p.enqueue(new, "Deleted")
	}
	return
}

func (p *podController) enqueue(obj interface{}, action string) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	klog.V(5).Infof("enqueue pod %s %s event", key, action)
	p.queue.Add(key)
}

func (p *podController) reconcile(key string) error {
	startTime := time.Now()
	klog.V(5).Infof("Started syncing Pod event %q (%v)", key, time.Since(startTime))
	defer func() {
		klog.V(5).Infof("Finished syncing Pod event %q (%v)", key, time.Since(startTime))
	}()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	pod, err := p.podInformer.Lister().Pods(namespace).Get(name)
	if err != nil {
		return err
	}

	deploy, err := utils.GetDeploymentByPod(pod, p.replicaSetLister, p.deploymentLister)
	if err != nil {
		return err
	}
	node, err := p.nodeLister.Get(pod.Spec.NodeName)
	if err != nil {
		return err
	}
	for _, policy := range p.policy.Spec.SyncAppPeriod {
		if policy.Period.Duration != 0 {
			activeDuration := policy.Period.Duration + utils.ExtraActivePeriod
			deployUsages, err := utils.GetResourceUsageRange(deploy.Annotations, policy.Name, activeDuration)
			if err != nil {
				klog.Warningf("get deployment %s/%s metrics %s from annotation failed: %v", deploy.Namespace, deploy.Name, policy.Name, err)
				continue
			}

			deltaName := utils.DeltaPrefixName + policy.Name
			deltaUsages, err := utils.GetDeltaUsageRange(node.Annotations, deltaName)
			if err != nil {
				klog.Warningf("get node %s metrics %s from annotation failed: %v", node.Name, deltaName, err)
				continue
			}

			var mergeType string
			if pod.DeletionTimestamp == nil {
				// scheduled pod event
				mergeType = utils.MergeTypeAdd
			} else {
				// delete pods event
				mergeType = utils.MergeTypeSub
			}
			deltaUsages, err = utils.MergeUsageRange(deltaUsages, deployUsages, mergeType)
			if err != nil {
				klog.Warningf("merge delta metrics %s and deployment metrics %s failed: %v", deltaName, policy.Name, err)
				continue
			}
			if err = utils.PatchNodeAnnotation(p.kubeClient, node, deltaName, deltaUsages); err != nil {
				klog.Warningf("patch node %s annotation %s failed: %v", node.Name, deltaName, err)
				return err
			}
		} else {
			klog.Warningf("syncAppPolicy %s duration value illegel", policy.Name)
			continue
		}
	}
	return nil
}
