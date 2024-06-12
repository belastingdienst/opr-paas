---
title: Configuring the operator
summary: A detailed description of the ConfigMap and its fields used to configure the operator.
authors:
  - hikarukin
date: 2024-07-09
---

Configuring the operator
========================

ConfigMap fields
----------------

This section is intended to create detailed documentation with regard to the
configmap and its fields that are used to configure the behaviour of the PAAS
operator.

### Top level fields

| Name                     | Type                | Description |
| ------------------------ | ------------------- | ----------- |
| argopermissions          | -                   | |
| applicationset_namespace | string              | |
| capabilities             | map                 | |
| debug                    | bool                | Turn on/off debugging. |
| decryptKeyPaths          | []string            | |
| exclude_appset_name      | string              | |
| ldap                     | -                   | |
| managed_by_label         | string              | |
| quota_label              | string              | |
| requestor_label          | string              | |
| rolemappings             | map[string][]string | |
| whitelist                | -                   | |

### Fields `argopermissions` level

The `argopermissions` level has the following underlying fields:

| Name                     | Type   | Description |
| ------------------------ | ------ | ----------- |
| header                   | string |             |
| resource_name            | string |             |
| retries                  | uint   |             |
| role                     | string |             |

### Map entry fields `capabilities` level

The `capabilities` field is a map of objects, each having:

| Name                     | Type   | Description |
| ------------------------ | ------ | ----------- |
| applicationset           | string | |
| quotas                   |  ConfigQuotaSettings      | |
| extra_permissions        |  ConfigCapPerm      | |
| default_permissions      |   ConfigCapPerm     | |

### Fields `ldap` level

| Name                     | Type   | Description |
| ------------------------ | ------ | ----------- |
|   host                   | string |             |
|   port                   | int32  |             |

### Fields `whitelist` level

| Name                     | Type   | Description |
| ------------------------ | ------ | ----------- |
| namespace                | string |             |
| name                     | string |             |

Example ConfigMap
-----------------

```yml
kind: ConfigMap
apiVersion: v1
metadata:
  name: opr-paas-config
  namespace: prod-paas
data:
  config.yaml: |
    ---
    debug: false
    whitelist:
      namespace: chp-cronjobs
      name: caaswhitelist
    ldap:
      host: ${PROD_LDAP_HOST}
      port: 636
    argopermissions:
      resource_name: argo-service
      role: admin
      header: |
        g, system:cluster-admins, role:admin
        g, something_clusteradmin, role:admin
        g, something, role:admin
      retries: 10
    managed_by_label: argocd.argoproj.io/managed-by
    requestor_label: level-one-support
    decryptKeyPaths:
      - /secrets/paas
    applicationset_namespace: prod-argocd
    exclude_appset_name: something-to-be-excluded
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
        quotas:
          defaults:
            limits.cpu: "8"
            limits.memory: 8Gi
            requests.cpu: "4"
            requests.memory: 5Gi
            requests.storage: "5Gi"
            thin.storageclass.storage.k8s.io/persistentvolumeclaims: "0"
        default_permissions:
          argocd-argocd-application-controller:
            - monitoring-edit
            - alert-routing-edit
      grafana:
        applicationset: prod-paas-grafana
        quotas:
          defaults:
            limits.cpu: "2"
            limits.memory: 3Gi
            requests.cpu: "1"
            requests.memory: 1Gi
            requests.storage: "2Gi"
            thin.storageclass.storage.k8s.io/persistentvolumeclaims: "0"
      tekton:
        applicationset: prod-paas-tekton
        quotas:
          clusterwide: true
          ratio: 0.1
          defaults:
            limits.cpu: "5"
            limits.memory: 8Gi
            requests.cpu: "1"
            requests.memory: 2Gi
            requests.storage: "100Gi"
            thin.storageclass.storage.k8s.io/persistentvolumeclaims: "0"
        default_permissions:
          pipeline:
            - monitoring-edit
            - alert-routing-edit
      sso:
        applicationset: prod-paas-sso
        quotas:
          defaults:
            limits.cpu: "4"
            limits.memory: 4Gi
            requests.cpu: "2"
            requests.memory: 2Gi
            requests.storage: "5Gi"
            thin.storageclass.storage.k8s.io/persistentvolumeclaims: "0"
```