//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1beta1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DynamicArgs) DeepCopyInto(out *DynamicArgs) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DynamicArgs.
func (in *DynamicArgs) DeepCopy() *DynamicArgs {
	if in == nil {
		return nil
	}
	out := new(DynamicArgs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DynamicArgs) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeResourceTopologyMatchArgs) DeepCopyInto(out *NodeResourceTopologyMatchArgs) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.TopologyAwareResources != nil {
		in, out := &in.TopologyAwareResources, &out.TopologyAwareResources
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeResourceTopologyMatchArgs.
func (in *NodeResourceTopologyMatchArgs) DeepCopy() *NodeResourceTopologyMatchArgs {
	if in == nil {
		return nil
	}
	out := new(NodeResourceTopologyMatchArgs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NodeResourceTopologyMatchArgs) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationResourceAwareArgs) DeepCopyInto(out *ApplicationResourceAwareArgs) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationResourceAwareArgs.
func (in *ApplicationResourceAwareArgs) DeepCopy() *ApplicationResourceAwareArgs {
	if in == nil {
		return nil
	}
	out := new(ApplicationResourceAwareArgs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ApplicationResourceAwareArgs) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}