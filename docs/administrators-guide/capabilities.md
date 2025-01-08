---
title: Configuring capabilities
summary: A detailed description of capabilities what it can do and how you can configure them in the PaasConfig CRD.
authors:
  - Devotional Phoenix
date: 2024-12-09
---

# Configuring capabilities

The Paas Operator can deliver capabilities to enable Paas deployments with CI and CD options with a one-click option.
Some examples of capabilities include:

- enabling ArgoCD for Continuous Delivery on your Paas namespaces
- enabling tekton for Continuous Integration of your application components
- observing your Paas resources with Grafana
- configuring federated Authentication and Authorization with keycloak

Adding capabilities does not require code changes / building new images. It only requires:

1. a git repository for the cluster-wide ArgoCD to be used for deploying the capability for a Paas which has the capability enabled
2. an ApplicationSet in the namespace of the cluster-wide ArgoCD to point to the git repository
3. configuration for the Paas operator pointing to the ApplicationSet and specifying default quota to be used when no quota is specified for this capability in the Paas

## Configuring the ApplicationSet

Cluster administrators can configure the ApplicationSet to be used for this specific capability.
Imagine a cluster-wide ArgoCD to manage capabilities for Paas'es.
It is deployed in the namespace `paas-capabilities-argocd`.
To enable any capability, `spec.clusterwide_argocd_namespace` needs to be set to `paas-capabilities-argocd`, so that the Paas operator will locate ApplicationSets for capabilities in this namespace.
And for a new capability (e.a. `new-capability`), there should be an ApplicationSet to manage resources for this new capability.
This ApplicationSet should be created in `paas-capabilities-argocd`, and it's name (e.a. `new-capability`) should be configured in PaasConfig (`spec.capabilities["new-capability"].ApplicationSet`).
After setting this configuration, for every Paas with the capability `new-capability` enabled, the Paas operator will
`GET` the ApplicationSet `paas-capabilities-argocd.new-capability`, add the Paas to the list generator and update the ApplicationSet definition..
This in turn will create a new Application for the capability for this Paas, and ArgoCD will create and manage the resources.

### Example ApplicationSet

!!! example

    ```yml
    apiVersion: argoproj.io/v1alpha1
    kind: ApplicationSet
    metadata:
      name: mycap-as
      namespace: paas-capabilities-argocd
    spec:
      generators: []
      template:
        metadata:
          name: '{{paas}}-capability-mycap'
        spec:
          destination:
            namespace: '{{paas}}-mycap'
            server: 'https://kubernetes.default.svc'
          project: '{{paas}}'
          source:
            kustomize:
              commonLabels:
                capability: mycap
                clusterquotagroup: '{{requestor}}'
                paas: '{{paas}}'
                service: '{{service}}'
                subservice: '{{subservice}}'
                key: '{{my-custom-key}}'
                revision: '{{my-custom-revision}}'
            path: paas-capabilities/mycap
            repoURL: 'https://www.github.com/belastingdienst/opr-paas-capabilities.git'
            targetRevision: main
          syncPolicy:
            automated:
              selfHeal: true
    ```

## Configuring quota

For every Capabilities for every Paas, a separate ClusterResourceQuota is created.
Quotas can be set in a Paas, and when not set, the Capability configuration can have a Default which will be used instead.
Furthermore, the capability configuration can also have a min and max value set.
The Paas operator will use the value as set in the Paas, and these Default, Min and Max settings to come to the proper value to be set in the ClusterResourceQuota set on the namespace.
Beyond these options, a capability can also be configured to use cluster-wide Quota with the `spec.capabilities["new-capability"].quotas.clusterwide`
and `spec.capabilities["new-capability"].quotas.raio`.

### More info

For more information please check:

- [administrators-guide's Cluster Wide Quotas section](./cluster-wide-quotas/basic-usage.md)
- [api-guide on capability quota configuration](../development-guide/00_api.md#configcapability)
- [api-guide on capability quota in the Paas](../development-guide/00_api.md#paascapability)

## Configuring permissions

For every capability the Paas operator can grant permissions to service accounts.
There are two options:

- Default permissions: These permissions are granted for this capability for every Paas
- Extra permissions: These permissions are granted only when a Paas has set `spec.capabilities[capability].extra_permissions` to true
  The main goal for extra permissions is to start off with higher permissions to get started, and revert them when a lower permissive option is available (e.a. lower permissions are set as default permissions).
  Customers starting with extra permissions can test with default permissions and return to extra permissions if they run into issues.

### More info

For more information on Default permissions and Extra permissions please revert to:

- [Example PaasConfig with a capability](#example-paasconfig-with-a-capability)
- [api-guide on capability configuration in the PaasConfig](../development-guide/00_api.md#configcapability)
- [api-guide on capability configuration in the Paas](../development-guide/00_api.md#paascapability)

## Configuring custom fields

Capabilities might require options to be set in a Paas. The fields to be set would be specific to a capability.
Some examples include:

- setting a git url, revision and path for a ArgoCD bootstrap application
- setting a version for the keycloak capability
- deploying multiple streams of a capability and allowing some DevOps teams to run a `latest` while others run a `stable` stream

For this reason we have introduced options for setting custom fields in the capability configuration in PaasConfig.
Each custom field belongs to a capability (e.a. `capability_name`), has a field name (e.a. `custom_field_name`) and configuration.
A custom field can be defined in PaasConfig in `PaasConfig.spec.capabilities[capability].customfields`).

The following configuration can be set:

- validation: A regular expression used to validate input in a Paas
- required: When set to true, an error is returned when the custom field is not defined in a Paas
- default: When set, a Paas without the custom field set will use this default instead.

\*\* note: `required` and `default` are mutually exclusive.

When set, a Paas can set these custom_fields, which brings them to the generators field in the Application created by the ApplicationSet for this specific Paas.

### Example of how a custom field operates

Image than on a cluster with

- a PaasConfig as defined in [Example PaasConfig with a capability](#example-paasconfig-with-a-capability), and
- an ApplicationSet as defined in [Example capability ApplicationSet](#example-ApplicationSet),
  a DevOps engineer created a Paas with a content like:

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: Paas
    metadata:
      name: my-paas
    spec:
      capabilities:
        mycap:
          custom_fields:
            my-custom-key: key_123
      ...
    ```

The following would happen:

- The Paas operator would check all custom_fields and use the following field values:
  - my-custom-key: key_123
  - my-custom-revision main
- The Paas operator would create an entry in the list generator in the ApplicationSet, with custom fields set as elements
- The cluster-wide ArgoCD ApplicationSet controller would create a new application for my-paas-capability-mycap
- the new application would have the following set in `spec.source.kustomize.commonLabels`:
  - key: key_123
  - revision: main
- From here, kustomize could use these values to be set on all resources create by the cluster-wide ArgoCD for this capability for this Paas

### More info

For more information on Custom Fields please revert to:

- [Example PaasConfig with a capability](#example-paasconfig-with-a-capability)
- [Example capability ApplicationSet with custom_fields being set as commonLabels](#example-ApplicationSet)
- [api-guide on capability configuration in the PaasConfig](../development-guide/00_api.md#configcustomfield)
- [api-guide on capability configuration in the Paas](../development-guide/00_api.md#paascapability)

## Configuring capabilities in the PaasConfig

On every cluster running the Paas operator, a PaasConfig resource is defined.
This PaasConfig resource holds the specific configuration for the operator.
For each capability a chapter needs to be set in `spec.capabilities`. An example can be found below.
Furthermore, the Paas operator needs to know the namespace where to search for ApplicationSets managing the capability (`spec.clusterwide_argocd_namespace`).

### Example PaasConfig with a capability

Below example shows all configuration required to configure a capability.

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      clusterwide_argocd_namespace: paas-capabilities-argocd
      capabilities:
        mycap:
          ApplicationSet: mycap-as
          default_permissions:
            my-service-account:
              - my-cluster-role
          extra_permissions:
            my-extra-service-account:
              - my-extra-cluster-role
          custom_fields:
            my-custom-key:
              validation: '^key_[0-9]+$'
              required: true
            my-custom-revision:
              validation: '^(main|develop|feature-.*)$'
              default: main
          quotas:
            clusterwide: true
            defaults:
              limits.cpu: "8"
              limits.memory: 8Gi
              requests.cpu: "4"
              requests.memory: 5Gi
              requests.storage: "5Gi"
            min:
              limits.cpu: "1"
              limits.memory: 1Gi
              requests.cpu: "500Mi"
              requests.memory: 500Mi
            max:
              limits.cpu: "16"
              limits.memory: 16Gi
              requests.cpu: "16"
              requests.memory: 16Gi
              requests.storage: "10Gi"
              thin.storageclass.storage.k8s.io/persistentvolumeclaims: "0"
            ratio: 10
    ```
