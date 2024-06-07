package applicationresourceaware

import (
	"context"
	"fmt"

	"github.com/gocrane/crane-scheduler/pkg/plugins/apis/config"
	"github.com/gocrane/crane-scheduler/pkg/plugins/apis/policy"
	"github.com/gocrane/crane-scheduler/pkg/utils"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	applisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/klog/v2"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
)

var _ framework.FilterPlugin = &ApplicationResourceAware{}

const (
	// Name is the name of the plugin used in the plugin registry and configurations.
	Name = "ApplicationResourceAware"
)

// ApplicationResourceAware is a real application load-aware scheduler plugin.
type ApplicationResourceAware struct {
	handle           framework.FrameworkHandle
	schedulerPolicy  *policy.DynamicSchedulerPolicy
	deploymentLister applisters.DeploymentLister
	replicasetLister applisters.ReplicaSetLister
}

// Name returns name of the plugin.
func (ara *ApplicationResourceAware) Name() string {
	return Name
}

// New returns a ApplicationResourceAware Scheduler object.
func New(plArgs runtime.Object, h framework.FrameworkHandle) (framework.Plugin, error) {
	args, ok := plArgs.(*config.ApplicationResourceAwareArgs)
	if !ok {
		return nil, fmt.Errorf("want args to be of type ApplicationResourceAwareArgs, got %T.", plArgs)
	}

	schedulerPolicy, err := utils.LoadPolicyFromFile(args.PolicyConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduler policy from config file: %v", err)
	}

	ctx := context.TODO()
	informerFactory := informers.NewSharedInformerFactory(h.ClientSet(), 0)
	deployLister := informerFactory.Apps().V1().Deployments().Lister()
	rsLister := informerFactory.Apps().V1().ReplicaSets().Lister()

	klog.V(5).Infof("Start plugin %s InformerFactory", Name)
	informerFactory.Start(ctx.Done())
	informerFactory.WaitForCacheSync(ctx.Done())

	ara := &ApplicationResourceAware{
		schedulerPolicy:  schedulerPolicy,
		handle:           h,
		deploymentLister: deployLister,
		replicasetLister: rsLister,
	}

	return ara, nil
}
