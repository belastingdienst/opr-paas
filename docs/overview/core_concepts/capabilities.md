# Paas Capabilities

The whole idea of Paas is to create a multi tenancy solution which allows DevOps
teams to request a context for their project, which we like to call a 'Project as a Service',
e.a. Paas.

As part of the Paas we want to allow DevOps teams to consume capabilities 'As A Service'.
Examples of such capabilities are:

- ArgoCD (Continuous Delivery);
- Tekton (Continuous Integration);
- Grafana (Observability); and
- KeyCloak (Single Sign On);

However, the list could be longer in the future.

Every Paas could have one or more of such capabilities defined, which means they
get these capabilities with the required permissions. The DevOps team requires no
specific knowledge or permissions to use these capabilities.

They are managed by a cluster wide ArgoCD, which creates them according to the
standards as defined by the platform team. They would be deployed specifically for
their Paas and only usable in that context.

We want Product teams to be able to define which capabilities should be available,
and to have control over the components that comprise these capabilities.

Therefore, it is designed a follows:

- The available capabilities are all defined in the PaasConfig
  Per capability the following can be defined:
    - the default quota to be used when no quota is set in the Paas
    - if the cluster wide quota feature should be enabled for this capability
    - the ApplicationSet that can be reconfigured (new entry in the list generator)
      for each Paas with the capability defined
- For every Paas where the capability is defined, the Paas controller will create:
    - a [PaasNs](PaasNs.yaml)
    - an entry in the ApplicationSet List Generator which in creates a new Application,
      which in turn makes a cluster wide ArgoCD deployment read from the configured git
      repo and create the required resources in the namespace as required

## Example:

### PaasConfig

In the PaasConfig the following could be configured:

!!! example

    ```yaml
    spec:
      capabilities:
        # Config for the argocd capability
        argocd:
          # For every Paas with this capability defined, the list generator in the
          # paas-argocd ApplicationSet should be extended
          applicationset: paas-argocd
          # Quotas can be set in the Paas, but for these quotas there are defaults to
          # apply when not set in the Paas.
          quotas:
            clusterwide: false
            defaults:
              limits.cpu: "7"
              requests.cpu: "3"
            min: {}
            max: {}
            ratio: 0
          # For all Paas's with the argocd capability defined, by default also set
          # these permissions for the specified service account
          default_permissions:
            argocd-argocd-application-controller:
              - monitoring-edit
              - alert-routing-edit
          extra_permissions: {}
        # Config for the grafana capability
        grafana:
          # For every Paas with this capability defined, the list generator in the
          # paas-grafana ApplicationSet should be extended
          applicationset: paas-grafana
          # Quotas can be set in the Paas, but for these quotas there are defaults to
          # apply when not set in the Paas.
          quotas:
            clusterwide: false
            defaults:
              limits.cpu: "2"
              requests.cpu: "1"
            min: {}
            max: {}
            ratio: 0
          default_permissions: {}
          extra_permissions: {}
    ```

### Example ApplicationSet

The ArgoCD Applicationset could look like this:

!!! example

    ```yaml
    apiVersion: argoproj.io/v1alpha1
    kind: ApplicationSet
    metadata:
      # name of the applicationset, this can be used for Paas instances with the
      # argocd capability defined
      name: paas-argocd
      # Specify the namespace of a cluster wide ArgoCD. On OpenShift the openshift-gitops
      # namespace is meant to have the only cluster wide ArgoCD.
      namespace: openshift-gitops
    spec:
      # This list can be empty, but is required in the definition.
      # Note that the Paas operator will create and manage a list generator here. So
      # when managing this applicationset with the cluster wide ArgoCD requires setting
      # up resourceExclusions
      generators: []
      template:
        metadata:
          name: "{{paas}}-capability-argocd"
        spec:
          destination:
            namespace: "{{paas}}-argocd"
            server: "https://kubernetes.default.svc"
          project: "{{paas}}"
          source:
            kustomize:
              commonLabels:
                capability: argocd
                clusterquotagroup: "{{requestor}}"
                paas: "{{paas}}"
                service: "{{service}}"
                subservice: "{{subservice}}"
            path: paas-capabilities/argocd
            repoURL: "ssh://git@github.com/belastingdienst/my-paas-capabilities.git"
            targetRevision: master
          syncPolicy:
            automated:
              selfHeal: true
    ```

### Example Paas

This would mean that someone could create a Paas with a block like this:

!!! example

    ```yaml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: Paas
    metadata:
      name: my-paas
    spec:
      capabilities:
        argocd:
          custom_fields:
            # Bootstrap application to point to the root folder
            gitPath: .
            # Bootstrap application to point to the main branch
            gitRevision: main
            # Bootstrap application to point to this repo
            gitUrl: "ssh://git@github.com/belastingdienst/my-paas-repo.git"
        grafana:
          quota:
            limits.cpu: "5"
            limits.memory: "2Gi"
    ```

This would result in:

- a `my-paas-argocd` `ClusterResourceQuota` and a `my-paas-grafana` `ClusterResourceQuota`;
  - `my-paas-argocd` has default quotas as specified in the configuration;
     (`limits.cpu: "7"`, `requests.cpu: "3"`)
  - `my-paas-grafana` has `limits.cpu` overridden to "5", `requests.cpu` defaulting to "1" and `limits.memory` set to '2Gi';
- a namespace called `my-paas-argocd` linked to the `my-paas-argocd` `ClusterResourceQuota`;
- a namespace called `my-paas-grafana` linked to the `my-paas-grafana` `ClusterResourceQuota`;
  The `applicationset` has an extra entry for the namespace so that the cluster-wide
  ArgoCD will create a grafana deployment in this namespace.
