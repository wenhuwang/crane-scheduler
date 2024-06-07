package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DynamicArgs is the args struction of Dynamic scheduler plugin.
type DynamicArgs struct {
	metav1.TypeMeta `json:",inline"`
	// PolicyConfigPath specified the path of policy config.
	PolicyConfigPath string `json:"policyConfigPath"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeResourceTopologyMatchArgs holds arguments used to configure the NodeResourceTopologyMatch plugin.
type NodeResourceTopologyMatchArgs struct {
	metav1.TypeMeta `json:",inline"`
	// TopologyAwareResources represents the resource names of topology.
	TopologyAwareResources []string `json:"topologyAwareResources,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApplicationResourceAwareArgs is the args struction of ApplicationResourceAware scheduler plugin.
type ApplicationResourceAwareArgs struct {
	metav1.TypeMeta `json:",inline"`
	// PolicyConfigPath specified the path of policy config.
	PolicyConfigPath string `json:"policyConfigPath"`
}
