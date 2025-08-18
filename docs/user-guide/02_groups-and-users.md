---
title: Managing permissions
summary: A short overview of defining authorization for users and groups
authors:
  - hikarukin
  - devotional-phoenix-97
date: 2025-01-21
---

## Groups and Users

For every Paas it is possible to define which k8s groups have permissions on resources belonging to the Paas.
Additionally, Administrators can define [rolemappings](../overview/core_concepts/authorization.md#paasconfig),
and groups in a Paas can have these functional roles applied.

It is possible to manage group membership externally, with an LDAP sync solution based on `oc adm group sync`.

for now, it is also possible to have group membership managed by the Paas operator, by specifying users.
But, we are working towards getting rid of user management through Paas, relying only on externally managed groups.

For more information on authorization, please see [Core Concepts - Authorization](../overview/core_concepts/authorization.md).

!!! note

    When both an LDAP query and a list of users is defined, the LDAP query takes precedence
    above the users. The paas operator will, in that case, not create a group, relying on the `oc adm group sync` to manage it.

!!! example

    ```yaml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: Paas
    metadata:
      name: tst-tst
    spec:
      groups:
        example_group:
          query: >-
            CN=example_group,OU=example,OU=UID,DC=example,DC=nl
          # Apply edit permissions for users in this group ; see PaasConfig rolemappings for more info
          roles:
            - edit
        second_example_group:
          users:
            - jdsmith
          # Apply admin permissions for users in this group ; see PaasConfig rolemappings for more info
          roles:
            - admin
    ```