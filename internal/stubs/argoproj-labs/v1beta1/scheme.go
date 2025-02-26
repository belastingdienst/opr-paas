// See readme for more info, in short: we skip CRD creation and trigger deepcopy generation with the following markers.
// +kubebuilder:skip
// +kubebuilder:object:generate=true
package v1beta1

/*
Because of dependency issues we decided to use a stub instead of importing all dependencies
behind the original code of ArgoCD. This of course introduces other risks, which we need to mitigate,
meaning when we use extra features of ArgoCD, we should check that we still have all parts of their CRD in our stub.

More info here: https://argo-cd.readthedocs.io/en/stable/user-guide/import/
*/

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// SchemeGroup is the group of the API that we are stubbing here
	SchemeGroup = "argoproj.io"

	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: SchemeGroup, Version: "v1beta1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// addKnownTypes adds the set of types defined in this package to the supplied scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		SchemeGroupVersion,
		&ArgoCD{},
		&ArgoCDList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
