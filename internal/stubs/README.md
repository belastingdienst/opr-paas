# Stubs

Because of dependency issues we don't want to import all original data types from ArgoCD project and use stubs instead.
This of coarse introduces other risks, which we need to mitigate as much as possible.
Two main risks can be identified:

- Newer version of the API's might change parts of the structure.
  Since we only stubb'ed the parts we need, and we only Patch instead of Update, this should not be an issue until the old API versions are no longer shipped with the running version of ArgoCD.
  We will detect and fix such issues as part of upgrading supported ArgoCD versions with our integration tests and fix then as required.
- Since we only stubb'ed part of the structs, we might require parts of the structs which we have not added yet.
  If so, adding these parts to our stubs should be part of the change to develop capabilities which require these extra data parts.

More info here: https://argo-cd.readthedocs.io/en/stable/user-guide/import/

