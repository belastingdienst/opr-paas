package v1alpha2

// NamespacedName is an internal type that can be used by the PaasConfig sub resources to define namespaced resources.
type NamespacedName struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`
}
