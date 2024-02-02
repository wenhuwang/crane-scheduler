package annotator

import (
	"fmt"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appinformers "k8s.io/client-go/informers/apps/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	applisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	policy "github.com/gocrane/crane-scheduler/pkg/plugins/apis/policy"

	prom "github.com/gocrane/crane-scheduler/pkg/controller/prometheus"
)

// Controller is Controller for node && deployment annotator.
type Controller struct {
	nodeInformer       coreinformers.NodeInformer
	nodeInformerSynced cache.InformerSynced
	nodeLister         corelisters.NodeLister

	eventInformer       coreinformers.EventInformer
	eventInformerSynced cache.InformerSynced
	eventLister         corelisters.EventLister

	namespaceLister         corelisters.NamespaceLister
	namespaceInformerSynced cache.InformerSynced

	deploymentLister         applisters.DeploymentLister
	deploymentInformerSynced cache.InformerSynced

	kubeClient clientset.Interface
	promClient prom.PromClient

	policy         policy.DynamicSchedulerPolicy
	bindingRecords *BindingRecords
}

// NewController returns a Node Annotator object.
func NewAnnotator(
	nodeInformer coreinformers.NodeInformer,
	eventInformer coreinformers.EventInformer,
	namespaceInformer coreinformers.NamespaceInformer,
	deploymentInformer appinformers.DeploymentInformer,
	kubeClient clientset.Interface,
	promClient prom.PromClient,
	policy policy.DynamicSchedulerPolicy,
	bingdingHeapSize int32,
) *Controller {
	return &Controller{
		nodeInformer:             nodeInformer,
		nodeInformerSynced:       nodeInformer.Informer().HasSynced,
		nodeLister:               nodeInformer.Lister(),
		namespaceLister:          namespaceInformer.Lister(),
		namespaceInformerSynced:  namespaceInformer.Informer().HasSynced,
		deploymentLister:         deploymentInformer.Lister(),
		deploymentInformerSynced: deploymentInformer.Informer().HasSynced,
		eventInformer:            eventInformer,
		eventInformerSynced:      eventInformer.Informer().HasSynced,
		eventLister:              eventInformer.Lister(),
		kubeClient:               kubeClient,
		promClient:               promClient,
		policy:                   policy,
		bindingRecords:           NewBindingRecords(bingdingHeapSize, getMaxHotVauleTimeRange(policy.Spec.HotValue)),
	}
}

// Run runs node annotator.
func (c *Controller) Run(worker int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()

	eventController := newEventController(c)
	c.eventInformer.Informer().AddEventHandler(eventController.handles())

	nodeController := newNodeController(c)
	deploymentController := newDeploymentController(c)

	if !cache.WaitForCacheSync(stopCh, c.nodeInformerSynced, c.eventInformerSynced, c.namespaceInformerSynced, c.deploymentInformerSynced) {
		return fmt.Errorf("failed to wait for cache sync for annotator")
	}
	klog.Info("Caches are synced for controller")

	for i := 0; i < worker; i++ {
		go wait.Until(nodeController.Run, time.Second, stopCh)
		go wait.Until(eventController.Run, time.Second, stopCh)
		go wait.Until(deploymentController.Run, time.Second, stopCh)
	}

	go wait.Until(c.bindingRecords.BindingsGC, time.Minute, stopCh)

	nodeController.CreateMetricSyncTicker(stopCh)
	deploymentController.CreateMetricSyncTicker(stopCh)

	<-stopCh
	return nil
}
