---
title: Utilizing capabilities
summary: A short overview of how to use capabilities.
authors:
  - hikarukin
  - devotional-phoenix-97
date: 2025-01-20
---

## Capabilities

One of the core features of the Paas operator is to enable Paas users with capabilities.
Capabilities need to be created and added to the cluster wide configuration of the Paas operator by administrators.
After that Paas users can easily add the capabilities to their Paas.

Read more about Paas capabilities in our [core concepts](../overview/core_concepts/capabilities.md) documentation.

!!! example

    ```yaml
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: Paas
    metadata:
      name: tst-tst
    spec:
      capabilities:
        argocd:
          enabled: true
          custom_fields
            gitPath: environments/production
            gitRevision: main
            gitUrl: >-
              ssh://git@git.example.nl/example/example-repo.git
        grafana:
          enabled: true
        sso:
          enabled: true
          quota:
            limits.cpu: '5'
            limits.memory: 8Gi
            requests.cpu: '2'
            requests.memory: 2Gi
            requests.storage: 100Gi
        tekton:
          enabled: true
          quota:
            limits.cpu: '32'
            limits.memory: 32Gi
            requests.cpu: '16'
            requests.memory: 16Gi
            requests.storage: 40Gi
    ```
