# OpenShift

The whole idea we tried to solve with the Paas operator, is to create a multi tenancy
solution which allows DevOps teams to request a context for their project, which
we like to call a 'Project as a Service', e.a. Paas.

This aligns heavily with large clusters servicing multiple DevOps teams, which
aligns closely with how we see other organizations running OpenShift.

For other deployments we mostly see small (nearly vanilla) K8S deployments, where
each cluster is only servicing one Devops team specifically. However, we do also
feel that having a single interface to consume features like user management,
capabilities, and quota management could be helpful to have in non-OpenShift
environments too.

## OpenShift specific dependencies

We rely on OpenShift for the following features:

- Cluster Wide Quotas, which seems to be built into the core of OpenShift and does
  not seem to have a k8s generic alternative. Running on vanilla K8S, we would
  probably leave options to have one quota for multiple namespaces and implement
  normal ResourceQuota definitions instead.
- We currently rely on the Groups implementation in OpenShift. We are revisiting
  the current architecture and will work towards a solution that can work natively
  in K8S as good as possible.

## ArgoCD integrations

### ArgoCD ApplicationSet List Generators.

The Paas operator has the option to create [capabilities](capabilities.md) and to
keep the implementations of capabilities freely programmable we have integrated
the Paas operator with ArgoCD.

This means that the expectations of running the Paas operator with capabilities,
is that there is a cluster wide ArgoCD deployment available, and for each capability
there is an additional ApplicationSet to manage Paas capabilities.

The Paas operator integrates through these capabilities by managing a list generator
in the ApplicationSet, which in turn creates an ArgoApplication for every Paas with 
the capability enabled.