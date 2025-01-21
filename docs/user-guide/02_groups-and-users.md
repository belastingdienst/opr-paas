---
title: Managing permissions
summary: A short overview of defining authorization for users and groups
authors:
  - hikarukin
  - devotional-phoenix-97
date: 2025-01-21
---

## Groups and Users

For every Paas it is possible to define which k8s groups have permissions on resources
belonging to the Paas. It is possible to manage group membership externally, but it is
also possible to have group membership managed by the Paas operator, and even integrate
the Paas operator with a ldap sync solution based on `oc adm group sync`.

For more information on authorization, please see [Core Concepts - Authorization](../overview/core_concepts/authorization.md).

!!! Note

    When both an LDAP query and a list of users is defined, the LDAP query takes precedence
    above the users. This because `oc adm group sync` overwrites / errors when there are already
    users in the group which it believes it should create.

!!! example

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
        second_example_group:
          users:
            - jdsmith
    ```
