---
title: Validating field names
summary: Options to configure validation of field names
authors:
  - devotional-phoenix
date: 2025-02-27
---

# Introduction

Most fields in the CRD are checked directly with kubebuilder validations,
But some fields can be validated with a regular expression that is configurable through the PaasConfig:

Below snippet shows how validations can be configured for the complete set of vailable validations:

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      validations:
        paas:
          # (v1.12) Validate name of Paas
          name: "^[a-z0-9-]*$"
          # (v1.12) Validate name of groups in paas
          groupName: "^[a-z0-9-]*$"
          # (v1.12) Validate name of namespaces in paas
          namespaceName: "^[a-z0-9-]*$"
          # (v1.12) Validate requestor field in paas
          requestor: "^[a-z0-9-]*$"
          # (v3.6) Allow quotas (by default all is allowed, by setting a regular expressions you can limit 
          # allowed quotas). Below example disallows limits.cpu (a.o.) and only allows the 4 types as stated.
          # This option has effect on:
          # - Quotas in a Paas resource
          # - Quotas in PaasConfig.spec.capabilities[*].quotas
          # - Quotas in PaasConfig.spec.maxAllowedSubmittedQuota.maxQuota
          allowedQuotas: "^(limits.memory|requests.cpu|requests.memory|requests.storage)$"
        paasConfig:
          # (v1.12) Validate name of capability in config
          capabilityName: "^[a-z0-9-]*$"
        paasNs:
          # (v1.12) Validate name of paasNs
          name: "^[a-z0-9-]*$"
    ...
    ```

!!! note

    If only one of `PaasConfig.spec.validations.paas.namespaceName`, and `PaasConfig.validations.paasNs.name` is set,
    both PaasNs names and Paas.Spec.Namespaces are validated with the same validation rule.

!!! tip "See Also"
    For details on how to enforce maximum limits on these quotas, see the [Max Allowed Submitted Quota](max-allowed-submitted-quota.md) guide.

## Recommendations for multi-tenant clusters

When different teams (tenants) can create Paas resources, the following configuration and behavior is relevant
for keeping tenants apart.

### Paas names

All namespaces of a Paas are named `<paas name>-<suffix>`. When the name of one Paas followed by a dash is a prefix
of the name of another Paas (e.g. `team` and `team-x`), both could claim the same namespace (`team` with namespace
`x-prod`, and `team-x` with namespace `prod`, both claim `team-x-prod`).

- The Paas webhook rejects the creation of a Paas whose name has such a prefix relation with an existing Paas.
- We recommend a name validation that makes this impossible altogether, e.g. by allowing exactly one dash:

    ```yml
    spec:
      validations:
        paas:
          name: "^([a-z][a-z0-9]{2,7}-[a-z0-9]+)$"
    ```

### Groups

- Groups with an LDAP `query` are named after the CN of the query. The Paas webhook rejects a group whose name is
  already used by another Paas for a different group (e.g. `CN=admins,OU=team-a` and `CN=admins,OU=team-b`). Using
  the exact same query in multiple Paases is allowed: that is the same LDAP group.
- Groups with `users` let Paas editors decide who gets access. In multi-tenant clusters we recommend to only allow
  groups with an LDAP query, by setting `spec.feature_flags.group_user_management` to `block` in the PaasConfig.

### Namespaces

The operator considers every namespace defined by a Paas (or a PaasNS belonging to it) as part of that Paas:
an existing namespace with that name is claimed (labels and owner reference are set), unless it is already owned
by another Paas. The operator is not meant to manage namespaces alongside manually created namespaces with names
that match a Paas; use a naming convention that keeps these apart.

The operator only deletes namespaces that are owned by the Paas, not namespaces that merely have its managed-by
label.

### Templates

Go templates in the PaasConfig are executed by the operator with the complete Paas and PaasConfig as context, and
their results end up in labels, custom fields and secrets that tenants can read. See
[Go templating](go-templating/introduction.md) for the functions that are not available, and only use values in
templates that may be shared with the tenants of the Paas.

