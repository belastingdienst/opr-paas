# GitOps-operator

As described in the docs, we integrate with the GitOps-operator.
In this way, we are able to deploy capabilities via a clusterwide `ArgoCD`.

## e2e-test

In order to run e2e-tests on vanilla k8s, we need to install the `ArgoCD crd`.
This CRD can be found in the accompanied [file](argoproj.io_applicationsets.yaml).

## LCM

While implementing the e2e-tests, we assumed the upstream CRD updates are backwards compatible.
Meaning, we assume we should be able to base the opr-paas on the current state of this CRD.
Ofcourse this is short-sighted, we will come up with a way to keep up with the upstream CRD and test
whether the e2e-tests succeed when a new release of the GitOps operator has been issued.

2025-04-08; we are exploring the options to get rid of the ArgoCD integration entirely by implementing
an ApplicationSet generator plugin.