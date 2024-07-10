# OpenShift

## Note

The whole idea we tried to solve with the PaaS operator, is to create a Multi Tenancy solution which allows DevOps teams to request a context for their project, which we like to call a 'Project as a Service', e.a. PaaS.
This aligns heavilly with large clusters servicing multiple DevOps teams, which aligns closely with how we see other organizations running OpenShift.
For other deployments we mostly see small (nearly vanilla) k8s deployments, where each cluster is only servicing one Devops team specifically.
But we do also feel that having a single interface to consume features like user management, capabilities, and quota management could be helpful to have in non-SopenSHift environments too.

## OpenShift specific dependencies

We rely on OpenShift for the following features:

- Clusterwide resource quota's, which seems to be built into the core of OpenShift and does not seem to have a k8s generic alternative.
  Running on vanilla k8s, we would probably leave options to have one quota for multiple namespaces and implement normal ResourceQuota definitions instead.
- We currenly rely on the Groups implementation in OpenShift. We are revisiting the current architecture and will work towards a solution that can work natively in k8s as good as posisble.

## ArgoCD integrations

### ArgoCD operator

The PaaS operator has the option to create [capabilities](capabilities.md) and for the argocd capability we integrate with the [argocd operator](https://github.com/argoproj-labs/argocd-operator).
This means that for the argocd capability, the PaaS operator creates the ClusterResourceQuota, Namespace, Permissions, etc. and creates an ArgoCD resource and a bootstrap Application (App of the Apps).

The [argocd-operator](github.com/argoproj-labs/argocd-operator) can be used on any K8s. On OpenShift it is known as the OpenShift gitops operator and is available through OLM.

### ArgoCD Applicationset List Generators.

The PaaS operator has the option to create [capabilities](capabilities.md) and to keep the implementations of capabilities freely programmable we have integrated the PaaS operator with ArgoCD.
This means that the expectations of running the PaaS operator with capabilities, is that there is a cluster-wide ArgoCD deployment available, and for each capability there is an additional APplicationSet to manage Paas capabilities.
The PaaS operator integrates through these capabilities by managing a list generator in the ApplicationSet, which in turn creates an ArgoApplication for every enabled capability in every enabled PaaS.
