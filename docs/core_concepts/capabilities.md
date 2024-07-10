# Paas Cabilities

The whole idea of PaaS is to create a Multi Tenancy solution which allows DevOps teams to request a context for their project, which we like to call a 'Project as a Service', e.a. PaaS.
As part of the PaaS we want to allow DevOps teams to consume capabilities 'As A Service'.
Examples of such capabilities are ArgoCD (Continuous Delivery), Tekton (Continuous Integration), Grafana (Observability) and KeyCloak (Single Sign On), but the list could be more.
Every PaaS could have one or more of such capabilities enabled, which means they get these capabilities with the required permissions.
The DevOps team requires no specific knowledge or permissions to use these capabilities.
They are managed by a cluster-wide ArgoCD, which creates them according to the standards as defined by the platform team.
And they would be deployed specificly for their PaaS and only usable in that context.
We want Product teams to be able to define which capabilities should be available, and to have control over the components that comprise these capabilities.

Therefore it is designed as:

- The available capabilities are all defined in the PaaS configuration
  Per capability the following can be defined:
  - the default quota to be used when no quota is set in teh PaaS
  - if the cluster-wide quota feature should be enabled for this capability
  - the applicationset that can be reconfigured (new entry in the list generator) for each Paas with the capability enabled
- For every PaaS where the capability is enabled, the PaaS controller will create
  - a [PaasNs](paasns.yaml)
  - an entry in the ApplicationSet List Generator which in creates a new Application, which in turn makes a cluster-wide ArgoCD deployment read from the configured git repo and create the required resources in the namespace as required

## Example:

### Paas Config

In the PaaS configuration the following could be configured:

```yaml
---
capabilities:
  # Config for the argocd capability
  argocd:
    # for every PaaS with this capability enabled, the list generator in the paas-argocd applicationset should be extended
    applicationset: paas-argocd
    # quota can be set in teh Paas, but for thesse quota's there are defaults to apply when not set in the PaaS.
    # note that
    defaultquotas:
      limits.cpu: "7"
      requests.cpu: "3"
    # For all PaaS's with the argocd capability enabled, by default also set these permissions for the specified service account
    default_permissions:
      argocd-argocd-application-controller:
        - monitoring-edit
        - alert-routing-edit
  # Config for the grafana capability
  grafana:
    # for every PaaS with this capability enabled, the list generator in the paas-grafana applicationset should be extended
    applicationset: paas-grafana
    # quota can be set in teh Paas, but for thesse quota's there are defaults to apply when not set in the PaaS.
    defaultquotas:
      limits.cpu: "2"
      requests.cpu: "1"
```

### Example ApplicationSet

The ArgoCD Applicationset could look like this:

```yaml
apiversion: argoproj.io/v1alpha1
kind: applicationset
metadata:
  # name of the applicationset, this can be used for PaaS instances with the argocd acapability enabled
  name: paas-argocd
  # Specify the namespace of a clusterwide ArgoCD. On OpenShift the openshift-gitops namespace is meant to have the only clusterwide ArgoCD.
  namespace: openshift-gitops
spec:
  # This list can be empty, but is required in the definition.
  # Note that the PaaS operator will create and manage a list generator here. So when managing this applicationset with the clusterwide ArgoCD requires setting up resourceExclusions
  generators: []
  template:
    metadata:
      name: "{{paas}}-cpet-capability-argocd"
    spec:
      destination:
        namespace: "{{paas}}-argocd"
        server: "https://kubernetes.default.svc"
      project: "{{paas}}"
      source:
        kustomize:
          commonlabels:
            capability: argocd
            clusterquotagroup: "{{requestor}}"
            paas: "{{paas}}"
            service: "{{service}}"
            subservice: "{{subservice}}"
        path: paas-capabilities/argocd
        repourl: "ssh://git@github.com/belastingdienst/my-paas-capabilities.git"
        targetrevision: master
      syncpolicy:
        automated:
          selfheal: true
```

### Example PaaS

This would mean that someone could create a Paas with a block like this:

```yaml
apiVersion: cpet.belastingdienst.nl/v1alpha1
kind: Paas
metadata:
  name: my-paas
spec:
  capabilities:
    argocd:
      enabled: true
      # Bootstrap application to point to the root folder
      gitPath: .
      # Bootstrap application to point to the main branch
      gitRevision: main
      # Bootstrap application to point to this repo
      gitUrl: "ssh://git@github.com/belastingdienst/my-paas-repo.git"
    grafana:
      enabled: true
      quota:
        limits.cpu: "5"
        limits.memory: "2Gi"
```

And that would result in:

- a my-paas-argocd ClusterResourceQuota and a my-paas-grafana ClusterResourceQuota.
  - my-paas-argocd has defaut quota's as specified in the config (limits.cpu: "7", requests.cpu: "3").
  - my-paas-grafana has limits.cpu: overridden to "5", requests.cpu defaulting to "1" and limits.memory set to '2Gi'.
- a my-paas namespace with a argocd PaasNs and a grafana PaasNs.
- a namespace called my-paas-argocd linked to the my-paas-argocd ClusterResourceQuota.
  The applicationset has an extra entry for the namespace so that the clusterwide ArgoCD will create a namespaced ArgoCD deployment in this namespace.
  The PaaS operator creates a bootstrap application which points to the root folder (.) of main branch of the ssh://git@github.com/belastingdienst/my-paas-repo.git repo.
- a namespace called my-paas-grafana linked to the my-paas-grafana ClusterResourceQuota.
  The applicationset has an extra entry for the namespace so that the clusterwide ArgoCD will create a grafana deployment in this namespace.
