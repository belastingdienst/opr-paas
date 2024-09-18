---
title: Adding extra components
summary: A description of the basic usage from the perspective of an end user.
authors:
  - hikarukin
date: 2024-07-04
---

## SSH Secrets

It is possible to define SSH secrets for your Paas's ArgoCD to use for access to
Github or BitBucket. They must be encrypted with the public key corresponding to
the private key that was deployed with the Paas operator.

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

## Groups and Users

It is possible to define groups in your Paas to allow access to the Paas' resources.
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

## Paas Quota

It is possible to request a specific quota for your Paas. This request will be
ignored if cluster wide resource quotas are configured by the administrators.

!!! Note
    Please note these will never overrule the maximum values configured by the
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

## Capabilities

It is possible to easily add certain capabilities to your Paas through the yaml
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

## Adding extra application namespaces

It is possible to define a list of extra namespaces to be created within the Paas.
These can be used for various purposes like dev, test and prod or for example a
team member's personal test.

These namespaces count towards the global quota requested by the Paas.

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
