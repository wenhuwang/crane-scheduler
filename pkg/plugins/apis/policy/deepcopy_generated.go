//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by deepcopy-gen. DO NOT EDIT.

package policy

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DynamicSchedulerPolicy) DeepCopyInto(out *DynamicSchedulerPolicy) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DynamicSchedulerPolicy.
func (in *DynamicSchedulerPolicy) DeepCopy() *DynamicSchedulerPolicy {
	if in == nil {
		return nil
	}
	out := new(DynamicSchedulerPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DynamicSchedulerPolicy) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HotValuePolicy) DeepCopyInto(out *HotValuePolicy) {
	*out = *in
	in.TimeRange.DeepCopyInto(&out.TimeRange)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HotValuePolicy.
func (in *HotValuePolicy) DeepCopy() *HotValuePolicy {
	if in == nil {
		return nil
	}
	out := new(HotValuePolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PolicySpec) DeepCopyInto(out *PolicySpec) {
	*out = *in
	if in.SyncPeriod != nil {
		in, out := &in.SyncPeriod, &out.SyncPeriod
		*out = make([]SyncPolicy, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.SyncAppPeriod != nil {
		in, out := &in.SyncAppPeriod, &out.SyncAppPeriod
		*out = make([]SyncPolicy, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Predicate != nil {
		in, out := &in.Predicate, &out.Predicate
		*out = make([]PredicatePolicy, len(*in))
		copy(*out, *in)
	}
	if in.Priority != nil {
		in, out := &in.Priority, &out.Priority
		*out = make([]PriorityPolicy, len(*in))
		copy(*out, *in)
	}
	if in.HotValue != nil {
		in, out := &in.HotValue, &out.HotValue
		*out = make([]HotValuePolicy, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PolicySpec.
func (in *PolicySpec) DeepCopy() *PolicySpec {
	if in == nil {
		return nil
	}
	out := new(PolicySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PredicatePolicy) DeepCopyInto(out *PredicatePolicy) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PredicatePolicy.
func (in *PredicatePolicy) DeepCopy() *PredicatePolicy {
	if in == nil {
		return nil
	}
	out := new(PredicatePolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PriorityPolicy) DeepCopyInto(out *PriorityPolicy) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PriorityPolicy.
func (in *PriorityPolicy) DeepCopy() *PriorityPolicy {
	if in == nil {
		return nil
	}
	out := new(PriorityPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SyncPolicy) DeepCopyInto(out *SyncPolicy) {
	*out = *in
	in.Period.DeepCopyInto(&out.Period)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SyncPolicy.
func (in *SyncPolicy) DeepCopy() *SyncPolicy {
	if in == nil {
		return nil
	}
	out := new(SyncPolicy)
	in.DeepCopyInto(out)
	return out
}
