package utils

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	applisters "k8s.io/client-go/listers/apps/v1"
)

const (
	TimeFormat       = "2006-01-02T15:04:05Z"
	DefaultTimeZone  = "Asia/Shanghai"
	DefaultNamespace = "crane-system"

	RangePrefix = "range"

	DeltaPrefixName = "delta_"
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

func GetDeploymentNameByPod(pod *corev1.Pod, rsLister applisters.ReplicaSetLister) (string, error) {
	for _, ownerRef := range pod.GetOwnerReferences() {
		if ownerRef.Kind == "ReplicaSet" {
			rs, err := rsLister.ReplicaSets(pod.Namespace).Get(ownerRef.Name)
			if err != nil {
				return "", err
			}
			for _, ownownerRef := range rs.GetOwnerReferences() {
				if ownownerRef.Kind == "Deployment" {
					return ownownerRef.Name, nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not find Deployment for pod %s", pod.Name)
}

func ParseRangeMetricsByString(str string) ([]float64, error) {
	var result []float64
	stringSlice := strings.Split(str, "|")
	for _, v := range stringSlice {
		fv, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, err
		}
		result = append(result, fv)
	}

	return result, nil
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
