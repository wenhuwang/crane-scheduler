# Crane-scheduler

## 新特性
- 避免热点调度问题：将计算热点调度逻辑集成到调度器进程内部从而解决了各种场景热点调度问题，包括但不限于：Deployment/Statefulset 管理的服务、Job 类型以及批量创建 Pod 类型业务
- 感知应用负载调度插件：根据 Deployment 历史平均 Pod 资源使用量和节点历史资源使用量来计算 Pod 是否适合调度到对应节点


## Overview
Crane-scheduler is a collection of scheduler plugins based on [scheduler framework](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/), including:

- [Dynamic scheduler: a load-aware scheduler plugin](doc/dynamic-scheduler.md)
## Get Started
### 1. Install Prometheus
Make sure your kubernetes cluster has Prometheus installed. If not, please refer to [Install Prometheus](https://github.com/gocrane/fadvisor/blob/main/README.md#prerequests).
### 2. Configure Prometheus Rules
Configure the rules of Prometheus to get expected aggregated data:
```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
    name: example-record
    labels:
        prometheus: k8s
        role: alert-rules
spec:
    groups:
    - name: node.rules
      interval: 1m
      rules:
      - record: cpu_usage_active
        expr: 1 - (avg by (cluster, instance) (irate(node_cpu_seconds_total{mode="idle"}[30s])))
      - record: mem_usage_active
        expr: 1-node_memory_MemAvailable_bytes/node_memory_MemTotal_bytes
      - record: cpu_usage_max_avg_1h
        expr: max_over_time(cpu_usage_avg_5m[1h])
      - record: cpu_usage_max_avg_1d
        expr: max_over_time(cpu_usage_avg_5m[1d])
      - record: cpu_usage_avg_5m
        expr: avg_over_time(cpu_usage_active[5m])
      - record: mem_usage_max_avg_1h
        expr: max_over_time(mem_usage_avg_5m[1h])
      - record: mem_usage_max_avg_1d
        expr: max_over_time(mem_usage_avg_5m[1d])
      - record: mem_usage_avg_5m
        expr: avg_over_time(mem_usage_active[5m])
      - record: load1_usage_avg_3m
        expr: label_replace(avg_over_time(node:load1:ratio[3m]), "instance", "$1", "node", "(.*)")
      - record: load5_usage_max_1h
        expr: label_replace(max_over_time(node:load5:ratio[1h]), "instance", "$1", "node", "(.*)")
      - record: load5_usage_max_1d
        expr: label_replace(max_over_time(node:load5:ratio[1d]), "instance", "$1", "node", "(.*)")
    - name: k8s.rules
      - expr: |
          sum by (namespace, deployment) (label_replace(namespace:workload_cpu_usage:sum{}, "deployment","$1","workload","Deployment:(.*)")) / sum by (namespace, deployment) (kube_deployment_spec_replicas{})
        record: pod_avg_cpu_usage
      - expr: |
          sum by (namespace, deployment) (label_replace(namespace:workload_memory_usage_wo_cache:sum{}, "deployment","$1","workload","Deployment:(.*)")) / sum by (namespace, deployment) (kube_deployment_spec_replicas{})
        record: pod_avg_mem_usage
      - expr: |
          sum (label_replace(label_join(sum(irate(container_cpu_usage_seconds_total{job="kubelet", pod!="", image!=""}[5m])) by (namespace,   pod, cluster) * on (pod, namespace) group_left(owner_kind,owner_name) label_join(label_replace(label_join(label_replace(label_replace  (kube_pod_owner{job="kube-state-metrics"},"owner_kind", "Deployment", "owner_kind", "ReplicaSet"), "owner_kind", "Pod", "owner_kind",   "<none>"),"tmp",":","owner_name","pod"),"owner_name","$1","tmp","<none>:(.*)"), "pod_name", "", "pod", "_name"), "workload",":",  "owner_kind","owner_name"), "workload","$1","workload","(Deployment:.+)-(.+)")) by (namespace, workload, owner_kind, cluster)
        record: namespace:workload_cpu_usage:sum
      - expr: |
          sum (label_replace(label_join(sum(container_memory_usage_bytes{job="kubelet", pod!="", image!=""} - container_memory_cache  {job="kubelet", pod!="", image!=""}) by (namespace, pod, cluster) * on (pod, namespace) group_left(owner_kind,owner_name) label_join  (label_replace(label_join(label_replace(label_replace(kube_pod_owner{job="kube-state-metrics"},"owner_kind", "Deployment",   "owner_kind", "ReplicaSet"), "owner_kind", "Pod", "owner_kind", "<none>"),"tmp",":","owner_name","pod"),"owner_name","$1","tmp",  "<none>:(.*)"), "pod_name", "", "pod", "_name"), "workload",":","owner_kind","owner_name"), "workload","$1","workload","(Deployment:.+)-(.+)")) by (namespace, workload, owner_kind, cluster)
        record: namespace:workload_memory_usage_wo_cache:sum
```
>**⚠️Troubleshooting:** The sampling interval of Prometheus must be less than 30 seconds, otherwise the above rules(such as cpu_usage_active) may not take effect.

### 3. Install Crane-scheduler
There are two options:
1) Install Crane-scheduler as a second scheduler:
   ```bash
   helm repo add crane https://gocrane.github.io/helm-charts
   helm install scheduler -n crane-system --create-namespace --set global.prometheusAddr="REPLACE_ME_WITH_PROMETHEUS_ADDR" crane/scheduler
   ```
2) Replace native Kube-scheduler with Crane-scheduler:
   1) Backup `/etc/kubernetes/manifests/kube-scheduler.yaml`
   ```bash
   cp /etc/kubernetes/manifests/kube-scheduler.yaml /etc/kubernetes/
   ```
   2) Modify configfile of kube-scheduler(`scheduler-config.yaml`) to enable Dynamic scheduler plugin and configure plugin args:
   ```yaml
   apiVersion: kubescheduler.config.k8s.io/v1beta2
   kind: KubeSchedulerConfiguration
   ...
   profiles:
   - schedulerName: default-scheduler
     plugins:
       filter:
         enabled:
         - name: Dynamic
       score:
         enabled:
         - name: Dynamic
           weight: 3
     pluginConfig:
     - name: Dynamic
        args:
         policyConfigPath: /etc/kubernetes/policy.yaml
   ...
   ```
   3) Create `/etc/kubernetes/policy.yaml`, using as scheduler policy of Dynamic plugin:
   ```yaml
    apiVersion: scheduler.policy.crane.io/v1alpha1
    kind: DynamicSchedulerPolicy
    spec:
      syncPolicy:
        ##cpu usage
        - name: cpu_usage_avg_5m
          period: 3m
        - name: cpu_usage_max_avg_1h
          period: 15m
        - name: cpu_usage_max_avg_1d
          period: 3h
        ##memory usage
        - name: mem_usage_avg_5m
          period: 3m
        - name: mem_usage_max_avg_1h
          period: 15m
        - name: mem_usage_max_avg_1d
          period: 3h

      predicate:
        ##cpu usage
        - name: cpu_usage_avg_5m
          maxLimitPecent: 0.65
        - name: cpu_usage_max_avg_1h
          maxLimitPecent: 0.75
        ##memory usage
        - name: mem_usage_avg_5m
          maxLimitPecent: 0.65
        - name: mem_usage_max_avg_1h
          maxLimitPecent: 0.75

      priority:
        ##cpu usage
        - name: cpu_usage_avg_5m
          weight: 0.2
        - name: cpu_usage_max_avg_1h
          weight: 0.3
        - name: cpu_usage_max_avg_1d
          weight: 0.5
        ##memory usage
        - name: mem_usage_avg_5m
          weight: 0.2
        - name: mem_usage_max_avg_1h
          weight: 0.3
        - name: mem_usage_max_avg_1d
          weight: 0.5

      hotValue:
        - timeRange: 5m
          count: 5
        - timeRange: 1m
          count: 2
   ```
   4) Modify `kube-scheduler.yaml` and replace kube-scheduler image with Crane-scheduler：
   ```yaml
   ...
    image: docker.io/gocrane/crane-scheduler:0.0.23
   ...
   ```
   1) Install [crane-scheduler-controller](deploy/controller/deployment.yaml):
    ```bash
    kubectl apply ./deploy/controller/rbac.yaml && kubectl apply -f ./deploy/controller/deployment.yaml
    ```

### 4. Schedule Pods With Crane-scheduler
Test Crane-scheduler with following example:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cpu-stress
spec:
  selector:
    matchLabels:
      app: cpu-stress
  replicas: 1
  template:
    metadata:
      labels:
        app: cpu-stress
    spec:
      schedulerName: crane-scheduler
      hostNetwork: true
      tolerations:
      - key: node.kubernetes.io/network-unavailable
        operator: Exists
        effect: NoSchedule
      containers:
      - name: stress
        image: docker.io/gocrane/stress:latest
        command: ["stress", "-c", "1"]
        resources:
          requests:
            memory: "1Gi"
            cpu: "1"
          limits:
            memory: "1Gi"
            cpu: "1"
```
>**Note:** Change `crane-scheduler` to `default-scheduler` if `crane-scheduler` is used as default.

There will be the following event if the test pod is successfully scheduled:
```bash
Type    Reason     Age   From             Message
----    ------     ----  ----             -------
Normal  Scheduled  28s   crane-scheduler  Successfully assigned default/cpu-stress-7669499b57-zmrgb to vm-162-247-ubuntu
```

## Compatibility Matrix

|  Scheduler Image Version       | Supported Kubernetes Version |
| ------------------------------ | :--------------------------: | 
|         0.0.23                 |        >=1.22.0              |
|         0.0.20                 |        >=1.18.0              | 

The default scheudler image version is `0.0.23`, and you can run the following command for quick replacement:

```bash
 KUBE_EDITOR="sed -i 's/v1beta2/v1beta1/g'" kubectl edit cm scheduler-config -n crane-system && KUBE_EDITOR="sed -i 's/0.0.23/0.0.20/g'" kubectl edit deploy crane-scheduler -n crane-system
```
