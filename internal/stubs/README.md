# Stubs
Because of depency issues we decided to use a stub instead of importing all dependencies behind the original code of ArgoCD.
This of coarse introduces other risks, which we need to mitigate, meaning when we use axtra features of ArgoCD,
we should check that we still have all parts of their CRD in our stub.

More info here: https://argo-cd.readthedocs.io/en/stable/user-guide/import/