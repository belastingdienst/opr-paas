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
belonging to the Paas. It is possible to manage group membership externally, with an LDAP sync solution based on `oc adm group sync`.
It is also possible to have group membership managed by the Paas operator, by specifying users. However, we are working towards getting rid of user management through Paas, relying only on externally managed groups.

For more information on authorization, please see [Core Concepts - Authorization](../overview/core_concepts/authorization.md).

!!! Note

    When both an LDAP query and a list of users is defined, the LDAP query takes precedence
    above the users. The paas operator will, in that case, no create a group, relying on the `oc adm group sync` to manage it.

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
