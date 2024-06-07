//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by conversion-gen. DO NOT EDIT.

package v1beta1

import (
	unsafe "unsafe"

	config "github.com/gocrane/crane-scheduler/pkg/plugins/apis/config"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*DynamicArgs)(nil), (*config.DynamicArgs)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1beta1_DynamicArgs_To_config_DynamicArgs(a.(*DynamicArgs), b.(*config.DynamicArgs), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*config.DynamicArgs)(nil), (*DynamicArgs)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_config_DynamicArgs_To_v1beta1_DynamicArgs(a.(*config.DynamicArgs), b.(*DynamicArgs), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*NodeResourceTopologyMatchArgs)(nil), (*config.NodeResourceTopologyMatchArgs)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1beta1_NodeResourceTopologyMatchArgs_To_config_NodeResourceTopologyMatchArgs(a.(*NodeResourceTopologyMatchArgs), b.(*config.NodeResourceTopologyMatchArgs), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*config.NodeResourceTopologyMatchArgs)(nil), (*NodeResourceTopologyMatchArgs)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_config_NodeResourceTopologyMatchArgs_To_v1beta1_NodeResourceTopologyMatchArgs(a.(*config.NodeResourceTopologyMatchArgs), b.(*NodeResourceTopologyMatchArgs), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*ApplicationResourceAwareArgs)(nil), (*config.ApplicationResourceAwareArgs)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1beta1_ApplicationResourceAwareArgs_To_config_ApplicationResourceAwareArgs(a.(*ApplicationResourceAwareArgs), b.(*config.ApplicationResourceAwareArgs), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*config.ApplicationResourceAwareArgs)(nil), (*ApplicationResourceAwareArgs)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_config_ApplicationResourceAwareArgs_To_v1beta1_ApplicationResourceAwareArgs(a.(*config.ApplicationResourceAwareArgs), b.(*ApplicationResourceAwareArgs), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1beta1_DynamicArgs_To_config_DynamicArgs(in *DynamicArgs, out *config.DynamicArgs, s conversion.Scope) error {
	out.PolicyConfigPath = in.PolicyConfigPath
	return nil
}

// Convert_v1beta1_DynamicArgs_To_config_DynamicArgs is an autogenerated conversion function.
func Convert_v1beta1_DynamicArgs_To_config_DynamicArgs(in *DynamicArgs, out *config.DynamicArgs, s conversion.Scope) error {
	return autoConvert_v1beta1_DynamicArgs_To_config_DynamicArgs(in, out, s)
}

func autoConvert_config_DynamicArgs_To_v1beta1_DynamicArgs(in *config.DynamicArgs, out *DynamicArgs, s conversion.Scope) error {
	out.PolicyConfigPath = in.PolicyConfigPath
	return nil
}

// Convert_config_DynamicArgs_To_v1beta1_DynamicArgs is an autogenerated conversion function.
func Convert_config_DynamicArgs_To_v1beta1_DynamicArgs(in *config.DynamicArgs, out *DynamicArgs, s conversion.Scope) error {
	return autoConvert_config_DynamicArgs_To_v1beta1_DynamicArgs(in, out, s)
}

func autoConvert_v1beta1_NodeResourceTopologyMatchArgs_To_config_NodeResourceTopologyMatchArgs(in *NodeResourceTopologyMatchArgs, out *config.NodeResourceTopologyMatchArgs, s conversion.Scope) error {
	out.TopologyAwareResources = *(*[]string)(unsafe.Pointer(&in.TopologyAwareResources))
	return nil
}

// Convert_v1beta1_NodeResourceTopologyMatchArgs_To_config_NodeResourceTopologyMatchArgs is an autogenerated conversion function.
func Convert_v1beta1_NodeResourceTopologyMatchArgs_To_config_NodeResourceTopologyMatchArgs(in *NodeResourceTopologyMatchArgs, out *config.NodeResourceTopologyMatchArgs, s conversion.Scope) error {
	return autoConvert_v1beta1_NodeResourceTopologyMatchArgs_To_config_NodeResourceTopologyMatchArgs(in, out, s)
}

func autoConvert_config_NodeResourceTopologyMatchArgs_To_v1beta1_NodeResourceTopologyMatchArgs(in *config.NodeResourceTopologyMatchArgs, out *NodeResourceTopologyMatchArgs, s conversion.Scope) error {
	out.TopologyAwareResources = *(*[]string)(unsafe.Pointer(&in.TopologyAwareResources))
	return nil
}

// Convert_config_NodeResourceTopologyMatchArgs_To_v1beta1_NodeResourceTopologyMatchArgs is an autogenerated conversion function.
func Convert_config_NodeResourceTopologyMatchArgs_To_v1beta1_NodeResourceTopologyMatchArgs(in *config.NodeResourceTopologyMatchArgs, out *NodeResourceTopologyMatchArgs, s conversion.Scope) error {
	return autoConvert_config_NodeResourceTopologyMatchArgs_To_v1beta1_NodeResourceTopologyMatchArgs(in, out, s)
}


func autoConvert_v1beta1_ApplicationResourceAwareArgs_To_config_ApplicationResourceAwareArgs(in *ApplicationResourceAwareArgs, out *config.ApplicationResourceAwareArgs, s conversion.Scope) error {
	out.PolicyConfigPath = in.PolicyConfigPath
	return nil
}

// Convert_v1beta1_ApplicationResourceAwareArgs_To_config_ApplicationResourceAwareArgs is an autogenerated conversion function.
func Convert_v1beta1_ApplicationResourceAwareArgs_To_config_ApplicationResourceAwareArgs(in *ApplicationResourceAwareArgs, out *config.ApplicationResourceAwareArgs, s conversion.Scope) error {
	return autoConvert_v1beta1_ApplicationResourceAwareArgs_To_config_ApplicationResourceAwareArgs(in, out, s)
}

func autoConvert_config_ApplicationResourceAwareArgs_To_v1beta1_ApplicationResourceAwareArgs(in *config.ApplicationResourceAwareArgs, out *ApplicationResourceAwareArgs, s conversion.Scope) error {
	out.PolicyConfigPath = in.PolicyConfigPath
	return nil
}

// Convert_config_ApplicationResourceAwareArgs_To_v1beta1_ApplicationResourceAwareArgs is an autogenerated conversion function.
func Convert_config_ApplicationResourceAwareArgs_To_v1beta1_ApplicationResourceAwareArgs(in *config.ApplicationResourceAwareArgs, out *ApplicationResourceAwareArgs, s conversion.Scope) error {
	return autoConvert_config_ApplicationResourceAwareArgs_To_v1beta1_ApplicationResourceAwareArgs(in, out, s)
}