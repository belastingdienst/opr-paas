---
title: Adding Application namespaces
summary: How to add application namespaces to a paas
authors:
  - hikarukin
  - devotional-phoenix-97
date: 2025-01-20
---

# Application namespaces

To deploy a (micro) service, usually a Paas would be extended by one or more namespaces.
Mostly these namespaces would be used for running the actual application components.
All application namespaces use a combined quota belonging specifically to this Paas.

## Setting Paas application namespace quota

Each Paas spec has a `required` field for specifying quota.
Each quota has a name referring to the exact [Resource Type](https://kubernetes.io/docs/concepts/policy/resource-quotas/#compute-resource-quota)
and has a value defined as a [k8s Resource Quantity](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity/).

This setting is applied to a Cluster Resource Quota which is applied to all application
namespaces created for this Paas.

!!! Note

    Capabilities have their own separate quotas which can be set from the capability block of a Paas.
    Capability quotas do not need to be included in the application quota.

!!! example

    ```yaml
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: Paas
    metadata:
      name: tst-tst
    spec:
      quota:
        limits.cpu: '40'
        limits.memory: 64Gi
        requests.cpu: '20'
        requests.memory: 32Gi
        requests.storage: 200Gi
    ```

## Adding namespaces in the Paas spec

It is possible to define a list of extra namespaces to be created within the Paas.
These can be used for various purposes like dev, test and prod or for example a
team member's personal test.

These namespaces count towards the global quota requested by the Paas.

!!! example

    ```yaml
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: Paas
    metadata:
      name: tst-tst
    spec:
      namespaces:
        - mark
        - tst
        - acceptance
        - prod
        - joel
    ```

## Adding PaasNs resources

Alternatively, a PaasNs resource could be added to a namespace belonging to the Paas.
Read more about this feature in [the PaasNs documentation](../overview/core_concepts/paasns.md).

!!! example

    ```yaml
    ---
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: PaasNs
    metadata:
      name: my-ns
      namespace: my-paas-argocd
    spec:
      Paas: tst-tst
    ```
