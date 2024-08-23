# Stubs

Because of dependency issues we don't want to import all original data types from ArgoCD project and use stubs instead.
This of course introduces other risks, which we need to mitigate as much as possible.
Two main risks can be identified:

- Newer version of the API's might change parts of the structure.
  Since we only stub the parts we need, and we only Patch instead of Update, this should not be an issue until the old API versions
  are no longer shipped with the running version of ArgoCD.
  We will detect and fix such issues as part of upgrading supported ArgoCD versions with our integration tests and fix then as
  required.
- Since we only stub part of the structs, we might require parts of the structs which we have not added yet.
  If so, adding these parts to our stubs should be part of the change to develop capabilities which require these extra data parts.

More info here: https://argo-cd.readthedocs.io/en/stable/user-guide/import/

## Generating manifests and deepcopies

As we only stub parts of the structs we need, we choose to generate the deepcopy methods by deepcopy-gen. This is assumed to be
more error-prone than altering the upstream zz_generated.deepcopy.go files to our needs.

We used the marker: `// +kubebuilder:object:generate:=true` to generate deepcopies. Those markers are placed at package level at
each stubbed package `scheme.go` file.

We used the marker: `// +kubebuilder:skip` to skip CRD generation as controller-gen would otherwise generate an empty CRD, which is
not something we need nor want. This is also the reason why we scrubbed the upstream structs and removed al other kubebuilder
markers.

Regenerating the deepcopies is triggered by executing `make generate`

More information about the markers can be found:

1. https://book.kubebuilder.io/reference/markers/object
2. https://book.kubebuilder.io/reference/markers/crd
