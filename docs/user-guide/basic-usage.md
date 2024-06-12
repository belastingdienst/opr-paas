---
title: Basic usage
summary: A description of the basic usage from the perspective of an end user.
authors:
  - hikarukin
date: 2024-07-04
---

Basic Usage
===========

Minimal PAAS managed by another PAAS
------------------------------------

Creating a configuration file to define a PAAS is fairly straight forward. The
configuration file should use the current API version `cpet.belastingdienst.nl/v1alpha1`
and define a `kind: Paas`.

The most minimal configuration requires at least a `name` in the `metadata` section
and either a capability `argocd` that is `enabled`, or a `managedByPaas` entry.

In the following example, we'll use the latter. The `managedByPaas` entry should
contain the name of the PAAS that is allowed to manage this PAAS.

Example PAAS definition being managed by another PAAS:

```yaml
---
apiVersion: cpet.belastingdienst.nl/v1alpha1
kind: Paas
metadata:
  name: tst-tst
spec:
  managedByPaas: trd-prt
```

Minimal PAAS, self-managed using ArgoCD
---------------------------------------

Example PAAS definition, using its own ArgoCD:

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

Adding extra components or functionality
----------------------------------------

### SSH Secrets

It is possible to define SSH secrets for your PAAS's ArgoCD to use for access to
Github or BitBucket. They must be encrypted with the public key corresponding to
the private key that was deployed with the PAAS operator.

SSH Secrets can be either defined on the generic `spec` level or underneath the
`argocd` capability.

Example:
```yaml
apiVersion: cpet.belastingdienst.nl/v1alpha1
kind: Paas
metadata:
  name: tst-tst
spec:
  sshSecrets:
    'ssh://git@my-git-host/my-git-repo.git': >-
      2wkeKe...g==
```

### Groups and Users

It is possible to define groups in your PAAS to allow access to the PAAS' resources.
These groups are filled with either an LDAP query and/or a list of users.

When both an LDAP query and a list of users is defined, the users from the list
are added in addition to the users from the LDAP group. If the user from the list
was already added through the LDAP group, the user is simply ignored.

Example:
```yaml
apiVersion: cpet.belastingdienst.nl/v1alpha1
kind: Paas
metadata:
  name: tst-tst
spec:
  groups:
    example_group:
      query: >-
        CN=example_group,OU=example,OU=UID,DC=example,DC=nl
      users:
        - jdsmith
    second_example_group:
      users:
        - jdsmith
```

### PAAS Quota

It is possible to request a specific quota for your PAAS. This request will be
ignored if cluster wide resource quotas are configured by the administrators.

**Please note:** these will never overrule the maximum values configured by the
                 administrators.

Example:
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

### Capabilities

It is possible to easily add certain capabilities to your PAAS through the yaml
configuration. For each capability you are also able to request a certain quota.

Example:
```yaml
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

### Adding extra application namespaces

It is possible to define a list of extra namespaces to be created within the PAAS.
These can be used for various purposes like dev, test and prod or for example a
team member's personal test.

These namespaces count towards the global quota requested by the PAAS.

Example:
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
