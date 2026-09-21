---
title: Capability configuration
summary: How to configure a capability in a PaasConfig
authors:
  - Devotional Phoenix
date: 2024-12-09
---

# Capability configuration

## Introduction
On every cluster running the Paas operator, a PaasConfig resource is defined.
This PaasConfig resource holds the specific configuration for the operator.
For each capability an entry needs to be set in `spec.capabilities` map. An example can be found below.
Furthermore, the Paas operator needs to know the namespace where to search for ApplicationSets managing the capability (`spec.clusterwide_argocd_namespace`).

## Example PaasConfig with a capability

Below example shows all configuration required to configure a capability.

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      clusterwide_argocd_namespace: paas-capabilities-argocd
      capabilities:
        mycap:
          applicationset: mycap-as
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
            ratio: 0.1
          secrets: |
            {{- $scrt := getPaasSecrets -}}
            {{- if gt (len $scrt) 0 -}}
              {{- $result := dict "paas-secrets" $scrt -}}
              {{- $result | toYAML -}}
            {{- end -}}
    ```
