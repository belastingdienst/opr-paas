//go:build !ignore_autogenerated

/*
Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ArgoCD) DeepCopyInto(out *ArgoCD) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ArgoCD.
func (in *ArgoCD) DeepCopy() *ArgoCD {
	if in == nil {
		return nil
	}
	out := new(ArgoCD)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ArgoCD) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ArgoCDList) DeepCopyInto(out *ArgoCDList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ArgoCD, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ArgoCDList.
func (in *ArgoCDList) DeepCopy() *ArgoCDList {
	if in == nil {
		return nil
	}
	out := new(ArgoCDList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ArgoCDList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ArgoCDRBACSpec) DeepCopyInto(out *ArgoCDRBACSpec) {
	*out = *in
	if in.Policy != nil {
		in, out := &in.Policy, &out.Policy
		*out = new(string)
		**out = **in
	}
	if in.Scopes != nil {
		in, out := &in.Scopes, &out.Scopes
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ArgoCDRBACSpec.
func (in *ArgoCDRBACSpec) DeepCopy() *ArgoCDRBACSpec {
	if in == nil {
		return nil
	}
	out := new(ArgoCDRBACSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ArgoCDSpec) DeepCopyInto(out *ArgoCDSpec) {
	*out = *in
	in.RBAC.DeepCopyInto(&out.RBAC)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ArgoCDSpec.
func (in *ArgoCDSpec) DeepCopy() *ArgoCDSpec {
	if in == nil {
		return nil
	}
	out := new(ArgoCDSpec)
	in.DeepCopyInto(out)
	return out
}
