package utils

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gocrane/crane-scheduler/pkg/plugins/apis/policy"
	corev1 "k8s.io/api/core/v1"
	applisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/klog/v2"
)

const (
	TimeFormat       = "2006-01-02T15:04:05Z"
	DefaultTimeZone  = "Asia/Shanghai"
	DefaultNamespace = "crane-system"

	RangePrefix = "range"

	DeltaPrefixName = "delta_"

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
		return "", fmt.Errorf("key[%s] not found", usedstr)
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

// inActivePeriod judges if node annotation with this timestamp is effective.
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
