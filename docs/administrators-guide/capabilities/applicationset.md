---
title: Capability ApplicationSet
summary: An example of the Applicationset to be used for custom capabilities
authors:
  - Devotional Phoenix
date: 2026-09-21
---

# Capability ApplicationSet

## Introduction

Cluster administrators can configure the ApplicationSet to be used for this specific capability.
Imagine a cluster-wide ArgoCD to manage capabilities for Paas'es.
It is deployed in the namespace `paas-capabilities-argocd`.
To enable any capability, `spec.clusterwide_argocd_namespace` needs to be set to `paas-capabilities-argocd`, so that the Paas operator will locate ApplicationSets for capabilities in this namespace.
And for a new capability (e.a. `new-capability`), there should be an ApplicationSet to manage resources for this new capability.
This ApplicationSet should be created in `paas-capabilities-argocd`, and it's name (e.a. `new-capability`) should be configured in PaasConfig (`spec.capabilities["new-capability"].ApplicationSet`).
After setting this configuration, for every Paas with the capability `new-capability` defined, the Paas operator will
`GET` the ApplicationSet `paas-capabilities-argocd.new-capability`, add the Paas to the list generator and update the ApplicationSet definition.
This in turn will create a new Application for the capability for this Paas, and ArgoCD will create and manage the resources.

## Example

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
