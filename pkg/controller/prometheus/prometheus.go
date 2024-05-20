package prometheus

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"k8s.io/klog/v2"
)

const (
	DefaultPrometheusQueryTimeout = 10 * time.Second
	defaultStep                   = 30 * time.Minute

	PrefixRange = "range_"
)

// PromClient provides client to interact with Prometheus.
type PromClient interface {
	// QueryByNodeIP queries data by node IP.
	QueryByNodeIP(string, string) (string, error)
	// QueryByNodeName queries data by node IP.
	QueryByNodeName(string, string) (string, error)
	// QueryByNodeIPWithOffset queries data by node IP with offset.
	QueryByNodeIPWithOffset(string, string, string) (string, error)
	// QueryByDeployment queries data by deployment.
	QueryByDeployment(string, string) (string, error)

	// QueryRangeByNodeIP queries range data by node IP.
	QueryRangeByNodeIP(string, string) (string, error)
	// QueryRangeByNodeName queries range data by node IP.
	QueryRangeByNodeName(string, string) (string, error)

	// QueryRangeByDeployment queries range data by deployment.
	QueryRangeByDeployment(string, string) (string, error)
}

type promClient struct {
	API v1.API
}

// NewPromClient returns PromClient interface.
func NewPromClient(addr string) (PromClient, error) {
	config := api.Config{
		Address: addr,
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &promClient{
		API: v1.NewAPI(client),
	}, nil
}

func (p *promClient) QueryByNodeIP(metricName, ip string) (string, error) {
	klog.V(6).Infof("Try to query %s by node IP[%s]", metricName, ip)

	querySelector := fmt.Sprintf("%s{instance=~\"%s\"}", metricName, ip)

	result, err := p.query(querySelector)
	if result != "" && err == nil {
		return result, nil
	}

	querySelector = fmt.Sprintf("%s{instance=~\"%s:.+\"}", metricName, ip)
	result, err = p.query(querySelector)
	if result != "" && err == nil {
		return result, nil
	}

	return "", err
}

func (p *promClient) QueryByNodeName(metricName, name string) (string, error) {
	klog.V(6).Infof("Try to query %s by node name[%s]", metricName, name)

	querySelector := fmt.Sprintf("%s{instance=~\"%s\"}", metricName, name)

	result, err := p.query(querySelector)
	if result != "" && err == nil {
		return result, nil
	}

	return "", err
}

func (p *promClient) QueryByNodeIPWithOffset(metricName, ip, offset string) (string, error) {
	klog.V(6).Info("Try to query %s with offset %s by node IP[%s]", metricName, offset, ip)

	querySelector := fmt.Sprintf("%s{instance=~\"%s\"} offset %s", metricName, ip, offset)
	result, err := p.query(querySelector)
	if result != "" && err == nil {
		return result, nil
	}

	querySelector = fmt.Sprintf("%s{instance=~\"%s:.+\"} offset %s", metricName, ip, offset)
	result, err = p.query(querySelector)
	if result != "" && err == nil {
		return result, nil
	}

	return "", err
}

func (p *promClient) QueryByDeployment(metricName, deployment string) (string, error) {
	klog.V(6).Infof("Try to query %s by deployment[%s]", metricName, deployment)

	querySelector := fmt.Sprintf("%s{deployment=\"%s\"}", metricName, deployment)
	result, err := p.query(querySelector)
	if result != "" && err == nil {
		return result, nil
	}

	return "", err
}

func (p *promClient) query(query string) (string, error) {
	klog.V(6).Infof("Begin to query prometheus by promQL [%s]...", query)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultPrometheusQueryTimeout)
	defer cancel()

	result, warnings, err := p.API.Query(ctx, query, time.Now())
	if err != nil {
		return "", err
	}

	if len(warnings) > 0 {
		return "", fmt.Errorf("unexpected warnings: %v", warnings)
	}

	if result.Type() != model.ValVector {
		return "", fmt.Errorf("illege result type: %v", result.Type())
	}

	var metricValue string
	for _, elem := range result.(model.Vector) {
		if float64(elem.Value) < float64(0) || math.IsNaN(float64(elem.Value)) {
			elem.Value = 0
		}
		metricValue = strconv.FormatFloat(float64(elem.Value), 'f', 5, 64)
	}

	return metricValue, nil
}

func (p *promClient) QueryRangeByNodeIP(metricName, ip string) (string, error) {
	klog.V(6).Infof("Try to query range %s by node IP[%s]", metricName, ip)

	querySelector := fmt.Sprintf("%s{instance=~\"%s\"}", metricName, ip)
	result, err := p.queryRange(querySelector)
	if result != "" && err == nil {
		return result, nil
	}

	querySelector = fmt.Sprintf("%s{instance=~\"%s:.+\"}", metricName, ip)
	result, err = p.queryRange(querySelector)
	if result != "" && err == nil {
		return result, nil
	}
	return "", err
}

func (p *promClient) QueryRangeByNodeName(metricName, name string) (string, error) {
	klog.V(6).Infof("Try to query range %s by node name[%s]", metricName, name)

	querySelector := fmt.Sprintf("%s{instance=~\"%s\"}", metricName, name)
	result, err := p.queryRange(querySelector)
	if result != "" && err == nil {
		return result, nil
	}
	return "", err
}

func (p *promClient) QueryRangeByDeployment(metricName, deployment string) (string, error) {
	klog.V(6).Infof("Try to query range %s by deployment[%s]", metricName, deployment)

	querySelector := fmt.Sprintf("%s{deployment=\"%s\"}", metricName, deployment)
	result, err := p.queryRange(querySelector)
	if result != "" && err == nil {
		return result, nil
	}

	return "", err
}

func (p *promClient) queryRange(query string) (string, error) {
	klog.V(6).Infof("Begin to query range prometheus by promQL [%s]...", query)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultPrometheusQueryTimeout)
	defer cancel()

	// 计算从昨天0点到今天0点时间值，请求这段时间的指标
	t1 := time.Now()
	t2 := t1.AddDate(0, 0, -1)
	todayTime := time.Date(t1.Year(), t1.Month(), t1.Day(), 0, 0, 0, 0, t1.Location())
	yesterdayTime := time.Date(t2.Year(), t2.Month(), t2.Day(), 0, 0, 0, 0, t2.Location())
	timeRange := v1.Range{
		Start: yesterdayTime,
		End:   todayTime,
		Step:  defaultStep,
	}
	result, warnings, err := p.API.QueryRange(ctx, query, timeRange)
	if err != nil {
		return "", err
	}

	if len(warnings) > 0 {
		return "", fmt.Errorf("unexpected warnings: %v", warnings)
	}

	if result.Type() != model.ValMatrix {
		return "", fmt.Errorf("illege result type: %v", result.Type())
	}

	var metricValues string
	if metrix, ok := result.(model.Matrix); ok && len(metrix) > 0 {
		for i, elem := range metrix[0].Values {
			if float64(elem.Value) < float64(0) || math.IsNaN(float64(elem.Value)) {
				elem.Value = 0
			}
			// 根据当前索引是否为最后一个元素决定字符串拼接逻辑
			if len(metrix[0].Values)-1 == i {
				metricValues = metricValues + strconv.FormatFloat(float64(elem.Value), 'f', 5, 64)
			} else {
				metricValues = metricValues + strconv.FormatFloat(float64(elem.Value), 'f', 5, 64) + "|"
			}

		}
	}

	if len(metricValues) == 0 {
		return "", fmt.Errorf("result is empty")
	}

	return metricValues, nil
}
