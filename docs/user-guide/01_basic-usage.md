---
title: Basic usage
summary: A description of the basic usage from the perspective of an end user.
authors:
  - hikarukin
date: 2024-07-04
---

Basic Usage
===========

Minimal Paas managed by another Paas
------------------------------------

Creating a configuration file to define a Paas is fairly straight forward. The
configuration file should use the current API version `cpet.belastingdienst.nl/v1alpha1`
and define a `kind: Paas`.

The most minimal configuration requires at least a `name` in the `metadata` section
and either a capability `argocd` that is `enabled`, or a `managedByPaas` entry.

In the following example, we'll use the latter. The `managedByPaas` entry should
contain the name of the Paas that is allowed to manage this Paas.

Example Paas definition being managed by another Paas:

```yaml
---
apiVersion: cpet.belastingdienst.nl/v1alpha1
kind: Paas
metadata:
  name: tst-tst
spec:
  managedByPaas: trd-prt
```

Minimal Paas, self-managed using ArgoCD
---------------------------------------

Example Paas definition, using its own ArgoCD:

```yaml
---
apiVersion: cpet.belastingdienst.nl/v1alpha1
kind: Paas
metadata:
  name: tst-tst
spec:
  capabilities:
    argocd:
      enabled: true
      gitPath: environments/production
      gitRevision: main
      gitUrl: >-
        ssh://git@git.example.nl/example/example-repo.git
```
