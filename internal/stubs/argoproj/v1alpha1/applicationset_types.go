/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true

// ApplicationSet is a set of Application resources
type ApplicationSet struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              ApplicationSetSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

// ApplicationSetSpec represents a class of application set state.
type ApplicationSetSpec struct {
	Generators []ApplicationSetGenerator `json:"generators" protobuf:"bytes,2,name=generators"`
}

// ApplicationSetGenerator represents a generator at the top level of an ApplicationSet.
type ApplicationSetGenerator struct {
	List *ListGenerator `json:"list,omitempty" protobuf:"bytes,1,name=list"`
}

// ListGenerator include items info
type ListGenerator struct {
	Elements []apiextensionsv1.JSON `json:"elements" protobuf:"bytes,1,name=elements"`
}

// +kubebuilder:object:root=true

// ApplicationSetList contains a list of ApplicationSet
type ApplicationSetList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []ApplicationSet `json:"items" protobuf:"bytes,2,rep,name=items"`
}
