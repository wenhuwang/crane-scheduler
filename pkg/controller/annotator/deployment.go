package annotator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gocrane/crane-scheduler/pkg/controller/prometheus"
	prom "github.com/gocrane/crane-scheduler/pkg/controller/prometheus"
	policy "github.com/gocrane/crane-scheduler/pkg/plugins/apis/policy"
	utils "github.com/gocrane/crane-scheduler/pkg/utils"
	"github.com/robfig/cron/v3"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type deploymentController struct {
	*Controller
	queue workqueue.RateLimitingInterface
}

func newDeploymentController(c *Controller) *deploymentController {
	deploymentRateLimiter := workqueue.NewItemExponentialFailureRateLimiter(DefaultBackOff,
		MaxBackOff)

	return &deploymentController{
		Controller: c,
		queue:      workqueue.NewNamedRateLimitingQueue(deploymentRateLimiter, "deployment_event_queue"),
	}
}

func (c *deploymentController) Run() {
	defer c.queue.ShutDown()
	klog.Infof("Start to reconcile deployment events")

	for c.processNextWorkItem() {
	}
}

func (c *deploymentController) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	forget, err := c.syncDeployment(key.(string))
	if err != nil {
		klog.Errorf("error syncing deployment metrics %q, requeuing: %v", key.(string), err)
		c.queue.AddRateLimited(key)
		return true
	}
	if forget {
		c.queue.Forget(key)
	}
	return true
}

func (c *deploymentController) syncDeployment(key string) (bool, error) {
	startTime := time.Now()
	klog.V(6).Infof("Started syncing deployment event %q (%v)", key, time.Since(startTime))
	defer func() {
		klog.V(6).Infof("Finished syncing deployment event %q (%v)", key, time.Since(startTime))
	}()

	namespace, metricsName, err := splitMetaKeyWithMetricName(key)
	if err != nil {
		return true, fmt.Errorf("invalid resource key: %s", key)
	}

	deployments, err := c.deploymentLister.Deployments(namespace).List(labels.Everything())
	if err != nil {
		return false, err
	}

	for _, deployment := range deployments {
		err := annotateDeploymentLoad(c.promClient, c.kubeClient, deployment, metricsName)
		if err != nil {
			return false, fmt.Errorf("failed to annotate deployment %s/%s: %v", deployment.Namespace, deployment.Name, err)
		}
	}

	return true, nil
}

func annotateDeploymentLoad(promClient prom.PromClient, kubeClient clientset.Interface, deployment *appsv1.Deployment, key string) error {
	var value string
	var err error
	if strings.HasPrefix(key, prometheus.PrefixRange) {
		metricName := strings.TrimPrefix(key, prometheus.PrefixRange)
		value, err = promClient.QueryRangeByDeployment(metricName, deployment.Name)
	} else {
		value, err = promClient.QueryByDeployment(key, deployment.Name)
	}
	if err == nil {
		return patchDeploymentAnnotation(kubeClient, deployment, key, value)
	}

	// 避免副本数为0或者新应用没有历史数据导致获取监控指标失败
	if *deployment.Spec.Replicas >= 1 && err.Error() != prometheus.ErrEmptyMsg {
		return fmt.Errorf("failed to get deployment %s metrics %s: %v", key, deployment.Name, err)
	}
	return nil
}

func patchDeploymentAnnotation(kubeClient clientset.Interface, deployment *appsv1.Deployment, key, value string) error {
	annotation := deployment.GetAnnotations()
	if annotation == nil {
		annotation = map[string]string{}
	}

	operator := "add"
	_, exist := annotation[key]
	if exist {
		operator = "replace"
	}

	patchAnnotationTemplate :=
		`[{
		"op": "%s",
		"path": "/metadata/annotations/%s",
		"value": "%s"
	}]`

	patchData := fmt.Sprintf(patchAnnotationTemplate, operator, key, value+","+utils.GetLocalTime())

	_, err := kubeClient.AppsV1().Deployments(deployment.Namespace).Patch(context.TODO(), deployment.Name, types.JSONPatchType, []byte(patchData), metav1.PatchOptions{})
	if err == nil {
		klog.V(6).Infof("Patch deployment %s/%s annotation %s value is %s", deployment.Namespace, deployment.Name, key, value)
	}
	return err
}

func (c *deploymentController) CreateMetricSyncTicker1(stopCh <-chan struct{}) {
	for _, p := range c.policy.Spec.SyncAppPeriod {
		enqueueFunc := func(policy policy.SyncPolicy) {
			namespaces, err := c.namespaceLister.List(labels.Everything())
			if err != nil {
				klog.Errorf("failed to list namespace: %v", err)
				return
			}

			for _, ns := range namespaces {
				c.queue.Add(handlingMetaKeyWithMetricName(ns.Name, policy.Name))
			}
		}

		enqueueFunc(p)

		go func(p policy.SyncPolicy) {
			ticker := time.NewTicker(p.Period.Duration)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					enqueueFunc(p)
				case <-stopCh:
					return
				}
			}
		}(p)
	}
}

type deploymentJob struct {
	Controller *deploymentController
	PolicyName string
}

func (d deploymentJob) Run() {
	namespaces, err := d.Controller.namespaceLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("failed to list namespace: %v", err)
		return
	}

	for _, ns := range namespaces {
		d.Controller.queue.Add(handlingMetaKeyWithMetricName(ns.Name, d.PolicyName))
	}
}

func (c *deploymentController) CreateMetricSyncTicker(stopCh <-chan struct{}) {
	klog.V(4).Infof("start sync deployment controller")

	cr := cron.New()
	for _, p := range c.policy.Spec.SyncAppPeriod {
		cr.AddJob("10 0 * * *", deploymentJob{
			Controller: c,
			PolicyName: p.Name,
		})
	}
	cr.Start()
}
