---
title: Configuring the operator
summary: A detailed description of the PaasConfig CRD and its fields used to configure the operator.
authors:
  - hikarukin
date: 2024-11-27
---

Configuring the operator
========================

The Paas Operator is configured using a Custom Resource Definition (CRD) called
PaasConfig.

PaasConfig
----------

Administrators can create a resource of kind PaasConfig in order to configure the
Paas Operator. The operator will only use a single instance and when adding
multiple PaasConfig instances, they will be ignored.

The operator will do its best to prevent incorrect configurations from being loaded
through a combination of CRD spec level validation and custom verification checks.

For details on the layout of a PaasConfig resource, please see the [development-guide's api section](../development-guide/00_api.md#paasconfig)
and more specifically the [section on PaasConfigSpec](../development-guide/00_api.md#paasconfigspec).

Alternatively, if you prefer, you could use [doc.crds.dev](https://doc.crds.dev/github.com/belastingdienst/opr-paas).

For an example, see below.

MaxAllowedSubmittedQuota (v1alpha2)
----------------------------------

!!! note
    `MaxAllowedSubmittedQuota` is only available in `PaasConfig` **v1alpha2**.

The operator can enforce an upper bound on the quota that users are allowed to submit in a `Paas` resource (`.spec.quota`).

This is configured via `PaasConfig.spec.maxAllowedSubmittedQuota.maxQuota`.

If a user submits a `Paas` with a quota value that is higher than the configured maximum for a given resource (for example `limits.cpu`), the request will be denied by the validating webhook.

Notes:

- The comparison is done per resource key (e.g. `limits.cpu`, `requests.memory`) using Kubernetes `resource.Quantity` semantics.
- The keys in `maxQuota` must match the allowed quota keys validation. Invalid keys are rejected when creating/updating the `PaasConfig`.
- This is a guardrail for submitted `Paas` quota requests. It does not replace cluster-level enforcement such as `ResourceQuota` / `LimitRange`.

Example PaasConfig
------------------

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      maxAllowedSubmittedQuota:
        maxQuota:
          limits.cpu: "8"
          limits.memory: 8Gi
          requests.cpu: "4"
          requests.memory: 5Gi
          requests.storage: "100Gi"
      validations:
        paas:
          groupNames: "^[a-z0-9-]*$"
      debug: false
      managed_by_label: argocd.argoproj.io/managed-by
      requestor_label: level-one-support
      decryptKeySecret:
        namespace: paas-system
        name: example-keys
      clusterwide_argocd_namespace: prod-argocd
      quota_label: clusterquotagroup
      rolemappings:
        default:
          - admin
        edit:
          - edit
        view:
          - view
        admin:
          - admin
      capabilities:
        argocd:
          applicationset: prod-paas-argocd
          default_permissions:
            argocd-argocd-application-controller:
              - monitoring-edit
              - alert-routing-edit
          custom_fields:
            git_url:
              validation: '^ssh:\/\/git@scm\/[a-zA-Z0-9-.\/]*.git$'
              required: true
            git_revision:
              default: main
            git_path:
              default: '.'
              validation: '^[a-zA-Z0-9.\/]*$'
          extra_permissions: {}
          quotas:
            clusterwide: false
            defaults:
              limits.cpu: "8"
              limits.memory: 8Gi
              requests.cpu: "4"
              requests.memory: 5Gi
              requests.storage: "5Gi"
              thin.storageclass.storage.k8s.io/persistentvolumeclaims: "0"
            min: {}
            max: {}
            ratio: 0
        grafana:
          applicationset: prod-paas-grafana
          default_permissions: {}
          extra_permissions: {}
          quotas:
            clusterwide: false
            defaults:
              limits.cpu: "2"
              limits.memory: 3Gi
              requests.cpu: "1"
              requests.memory: 1Gi
              requests.storage: "2Gi"
              thin.storageclass.storage.k8s.io/persistentvolumeclaims: "0"
            min: {}
            max: {}
            ratio: 0
        tekton:
          applicationset: prod-paas-tekton
          default_permissions:
            pipeline:
              - monitoring-edit
              - alert-routing-edit
          extra_permissions: {}
          quotas:
            clusterwide: true
            defaults:
              limits.cpu: "5"
              limits.memory: 8Gi
              requests.cpu: "1"
              requests.memory: 2Gi
              requests.storage: "100Gi"
              thin.storageclass.storage.k8s.io/persistentvolumeclaims: "0"
            min: {}
            max: {}
            ratio: 0.1
        sso:
          applicationset: prod-paas-sso
          default_permissions: {}
          extra_permissions: {}
          quotas:
            clusterwide: false
            defaults:
              limits.cpu: "4"
              limits.memory: 4Gi
              requests.cpu: "2"
              requests.memory: 2Gi
              requests.storage: "5Gi"
              thin.storageclass.storage.k8s.io/persistentvolumeclaims: "0"
            min: {}
            max: {}
            ratio: 0
    ```

!!! note
    .spec.decryptKeySecret.name points to a secret `example-keys`.
    An example secret can be found [here](../../examples/resources/example-keys.yaml).
    For generating a new secret, please checkout our [kubectl-paas](https://github.com/belastingdienst/opr-paas-cli/) tool.

Example denial:

- Configured max: `limits.cpu: "8"`
- User submits: `Paas.spec.quota.limits.cpu: "10"`
- Result: admission denied with an error similar to:

    `quota (limits.cpu) cannot be larger than MaxAllowedSubmittedQuota (8)`