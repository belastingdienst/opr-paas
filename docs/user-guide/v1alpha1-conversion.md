---
title: Converting v1alpha1 to v1alpha2
summary: A detailed description of how to convert a v1alpha1.Paas / PaasNS to v1alpha2.Paas / PaasNS.
authors:
  - Devotional Phoenix
date: 2025-07-10
---

# Introduction

With release v2 we also released a new api v1alpha2 which has a slightly changed definition.

## Changes

### Paas

The following has changed between v1alpha1.Paas and v1alpha2.Paas:

- The following fields are removed from the Paas Capabilities (Paas.Spec.Capabilities):

    - `enabled`: Remove field if set to true, or remove Capability when set to false
    - `gitPath`, `gitRevision`, `gitUrl`: Rewrite to custom fields
    - `sshSecrets`: Rename to `secrets`
    - `namespaces`: Rewrite from a list to a map

!!! example

    This example of a v1alpha1 Paas

    ```yaml
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: Paas
    metadata:
      name: tst-tst
    spec:
      capabilities:
        argocd:
          # this field should be removed
          enabled: true
          gitPath: .
          gitRevision: main
          gitUrl: https://www.github.com/my-org/my-repo/
          quota:
            limits.cpu: '32'
          # `sshSecrets` should be changed to `secrets`
          sshSecrets:
            'ssh://git@my-git-host/my-git-repo.git': >-
              2wkeKe...g==
        # `tekton` is disabled and should be removed
        tekton:
          enabled: false
      # `sshSecrets` should be changed to `secrets`
      sshSecrets:
        'ssh://git@my-git-host/my-git-repo.git': >-
          2wkeKe...g==
      # `namespaces` should be rewritten from list to map
      namespaces:
        - ns1
        - ns2
    ```

    Would be rewritten as:

    ```yaml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: Paas
    metadata:
      name: tst-tst
    spec:
      capabilities:
        argocd:
          quota:
            limits.cpu: '32'
          secrets:
            'ssh://git@my-git-host/my-git-repo.git': >-
              2wkeKe...g==
          custom_fields:
            gitPath: .
            gitRevision: main
            gitUrl: https://www.github.com/my-org/my-repo/
      secrets:
        'ssh://git@my-git-host/my-git-repo.git': >-
          2wkeKe...g==
      namespaces:
        ns1: {}
        ns2: {}
    ```

### PaasNS

The following has changed between v1alpha1 and v1alpha2 PaasNS:

- `sshSecrets`: Rename to `secrets`

!!! example

    This example of a v1alpha1 PaasNS

    ```yaml
    ---
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: PaasNS
    metadata:
      name: my-ns
      namespace: my-paas-argocd
    spec:
      # This field is not required in v1alpha2
      paas: tst-tst
      # sshSecrets should be rewritten to `secrets`
      sshSecrets:
        'ssh://git@my-git-host/my-git-repo.git': >-
          2wkeKe...g==
    ```

    Would be rewritten as follows

    ```yaml
    ---
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasNS
    metadata:
      name: my-ns
      namespace: my-paas-argocd
    spec:
      secrets:
        'ssh://git@my-git-host/my-git-repo.git': >-
          2wkeKe...g==
    ```