# OpenShift

As described in the docs, we integrate with OpenShift.
In this way, we are able to bootstrap OpenShift resources for PaaS customers.

As there is no lightweight container distribution of OpenShift to use during e2e-tests, we
use a vanilla k8s cluster and mock all OpenShift dependencies. This implies we can only validate
whether these mocked resources are created correctly, not the behaviour OpenShift applies to those resources.

## LCM

While starting on the e2e-tests, we assume the upstream CRD updates are backwards compatible.
Meaning, we assume we should be able to base the opr-paas on the current state of the CRDs.
Ofcourse this is short-sighted, we will come up with a way to keep up with the upstream CRD and test
whether the e2e-tests succeed when a new release of the GitOps operator has been issued.

The upstream CRD of ClusterResourceQuota can be found here:
https://github.com/openshift/api/blob/release-4.14/quota/v1/0000_03_quota-openshift_01_clusterresourcequota.crd.yaml

The Group resources however, is baked into the OpenShift API. Therefore, there is no CRD available which we can install.
As we can't mock / reproduce such thing, we choose to build our own CRD, based on the Group struct.

The upstream source of the Group struct can be found here:
https://github.com/openshift/api/blob/release-4.14/user/v1/types.go#L148