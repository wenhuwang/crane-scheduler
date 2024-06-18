package utils

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gocrane/crane-scheduler/pkg/plugins/apis/policy"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	applisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/klog/v2"
)

const (
	TimeFormat       = "2006-01-02T15:04:05Z"
	DefaultTimeZone  = "Asia/Shanghai"
	DefaultNamespace = "crane-system"

	DeltaPrefixName = "delta_"
	RangePrefix     = "range"

	MergeTypeAdd = "addition"
	MergeTypeSub = "subtraction"

	// MinTimestampStrLength defines the min length of timestamp string.
	MinTimestampStrLength = 5
	// ExtraActivePeriod gives extra active time to the annotation.
	ExtraActivePeriod = 5 * time.Minute
)

// IsDaemonsetPod judges if this pod belongs to one daemonset workload.
func IsDaemonsetPod(pod *corev1.Pod) bool {
	for _, ownerRef := range pod.GetOwnerReferences() {
		if ownerRef.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

func GetLocalTime() string {
	loc := GetLocation()
	if loc == nil {
		time.Now().Format(TimeFormat)
	}

	return time.Now().In(loc).Format(TimeFormat)
}

func GetLocation() *time.Location {
	zone := os.Getenv("TZ")

	if zone == "" {
		zone = DefaultTimeZone
	}

	loc, _ := time.LoadLocation(zone)

	return loc
}

func GetSystemNamespace() string {
	ns := os.Getenv("CRANE_SYSTEM_NAMESPACE")

	if ns == "" {
		ns = DefaultNamespace
	}

	return ns
}

// NormalizaScore nornalize the score in range [min, max]
func NormalizeScore(value, max, min int64) int64 {
	if value < min {
		value = min
	}

	if value > max {
		value = max
	}

	return value
}

func GetDeploymentByPod(pod *corev1.Pod, rsLister applisters.ReplicaSetLister, deployLister applisters.DeploymentLister) (*appsv1.Deployment, error) {
	for _, ownerRef := range pod.GetOwnerReferences() {
		if ownerRef.Kind == "ReplicaSet" {
			rs, err := rsLister.ReplicaSets(pod.Namespace).Get(ownerRef.Name)
			if err != nil {
				return nil, fmt.Errorf("get replicaSet by pod failed: %v", err)
			}
			for _, ownerRef := range rs.GetOwnerReferences() {
				if ownerRef.Kind == "Deployment" {
					deploy, err := deployLister.Deployments(pod.Namespace).Get(ownerRef.Name)
					if err != nil {
						return nil, fmt.Errorf("get deployment by replicaSet failed: %v", err)
					}
					return deploy, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("could not find Deployment for pod %s", pod.Name)
}

func GetResourceUsage(anno map[string]string, key string, activeDuration time.Duration) (float64, error) {
	usedstr, ok := anno[key]
	if !ok {
		return 0, fmt.Errorf("key[%s] not found", usedstr)
	}

	usedSlice := strings.Split(usedstr, ",")
	if len(usedSlice) != 2 {
		return 0, fmt.Errorf("illegel value: %s", usedstr)
	}

	if !InActivePeriod(usedSlice[1], activeDuration) {
		return 0, fmt.Errorf("timestamp[%s] is expired", usedstr)
	}

	UsedValue, err := strconv.ParseFloat(usedSlice[0], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse float[%s]", usedSlice[0])
	}

	if UsedValue < 0 {
		return 0, fmt.Errorf("illegel value: %s", usedstr)
	}

	return UsedValue, nil
}

func GetResourceUsageRange(anno map[string]string, key string, activeDuration time.Duration) (string, error) {
	usedstr, ok := anno[key]
	if !ok {
		return "", nil
	}

	usedSlice := strings.Split(usedstr, ",")
	if len(usedSlice) != 2 {
		return "", fmt.Errorf("illegel value: %s", usedstr)
	}

	if !InActivePeriod(usedSlice[1], activeDuration) {
		return "", fmt.Errorf("timestamp[%s] is expired", usedstr)
	}

	return usedSlice[0], nil
}

func InActivePeriod(updatetimeStr string, activeDuration time.Duration) bool {
	if len(updatetimeStr) < MinTimestampStrLength {
		klog.Errorf("[crane] illegel timestamp: %s", updatetimeStr)
		return false
	}
	originUpdateTime, err := time.ParseInLocation(TimeFormat, updatetimeStr, GetLocation())
	if err != nil {
		klog.Errorf("[crane] failed to parse timestamp: %v", err)
		return false
	}
	now, updatetime := time.Now(), originUpdateTime.Add(activeDuration)
	if now.Before(updatetime) {
		return true
	}
	return false
}

func GetActiveDuration(syncPeriodList []policy.SyncPolicy, name string) (time.Duration, error) {
	for _, period := range syncPeriodList {
		if period.Name == name {
			if period.Period.Duration != 0 {
				return period.Period.Duration + ExtraActivePeriod, nil
			}
		}
	}

	return 0, fmt.Errorf("failed to get the active duration")
}

func GetDeltaUsageRange(anno map[string]string, key string) (string, error) {
	deltaStr, ok := anno[key]
	if !ok {
		return "", nil
	}

	usedSlice := strings.Split(deltaStr, ",")
	if len(usedSlice) != 2 {
		return "", fmt.Errorf("illegel value: %s", deltaStr)
	}
	return usedSlice[0], nil
}

func ParseRangeMetricsByString(str string) ([]float64, error) {
	var result []float64
	if str != "" {
		stringSlice := strings.Split(str, "|")
		for _, v := range stringSlice {
			fv, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, err
			}
			result = append(result, fv)
		}
	}

	return result, nil
}

func MergeUsageRange(deltaStr string, deployStr string, mergeType string) (string, error) {
	if deltaStr == "" && mergeType == MergeTypeAdd {
		return deployStr, nil
	}

	deployUsages, err := ParseRangeMetricsByString(deployStr)
	if err != nil {
		return "", fmt.Errorf("parse deployment range metrics %s failed: %v", deployStr, err)
	}

	var deltaUsages []float64
	var mergeFlag bool
	if deltaStr != "" {
		deltaUsages, err = ParseRangeMetricsByString(deltaStr)
		if err != nil {
			return "", fmt.Errorf("parse node delta range metrics %s failed: %v", deltaStr, err)
		}

		if len(deltaUsages) > len(deployUsages) {
			return "", fmt.Errorf("delta usages length %d more than the deployment usages length %d", len(deltaUsages), len(deployUsages))
		} else if len(deltaUsages) == len(deployUsages) {
			mergeFlag = true
		}
	}

	if !mergeFlag {
		deltaUsages = make([]float64, len(deployUsages))
	}

	var result string
	if mergeType == MergeTypeAdd {
		for i, v := range deltaUsages {
			usage := v + deployUsages[i]
			if i < len(deltaUsages)-1 {
				result += fmt.Sprintf("%.5f|", usage)
			} else {
				result += fmt.Sprintf("%.5f", usage)
			}
		}
	} else if mergeType == MergeTypeSub {
		for i, v := range deltaUsages {
			usage := v - deployUsages[i]
			if i < len(deltaUsages)-1 {
				result += fmt.Sprintf("%.5f|", usage)
			} else {
				result += fmt.Sprintf("%.5f", usage)
			}
		}
	}
	return result, nil
}

func PatchNodeAnnotation(kubeClient clientset.Interface, node *v1.Node, key, value string) error {
	annotation := node.GetAnnotations()
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

	patchData := fmt.Sprintf(patchAnnotationTemplate, operator, key, value+","+GetLocalTime())

	if _, err := kubeClient.CoreV1().Nodes().Patch(context.TODO(), node.Name, types.JSONPatchType, []byte(patchData), metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("patch node failed: %v", err)
	}
	return nil
}

func DeleteNodeAnnotation(kubeClient clientset.Interface, node *v1.Node, key string) error {
	operator := "remove"
	patchAnnotationTemplate :=
		`[{
		"op": "%s",
		"path": "/metadata/annotations/%s"
	}]`
	patchData := fmt.Sprintf(patchAnnotationTemplate, operator, key)
	if _, err := kubeClient.CoreV1().Nodes().Patch(context.TODO(), node.Name, types.JSONPatchType, []byte(patchData), metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("patch node failed: %v", err)
	}
	return nil
}
