---
title: Converting v1alpha1 to v1alpha2
summary: A detailed description of how to convert a v1alpha1.PaasConfig to v1alpha2.PaasConfig.
authors:
  - Devotional Phoenix
date: 2025-07-09
---

# Introduction

With release v2 we also released a new api v1alpha2 which has a slightly changed definition.
The following has changed between v1alpha1.PaasConfig and v1alpha2.PaasConfig:

- The following endpoints where deprecated in v1alpha1 and have been removed in v1alpha2:
  - GroupSyncList, GroupSyncListKey, LDAP
    (we no longer manage the LDAP GroupSyncList implementation)
  - ArgoPermissions, ArgoEnabled, ExcludeAppSetName
    (these setting belong to a hardcoded implementation which is replaced by a more flexible implementation)
- A new label implementation replaces the following label options which are removed in v1alpha2
  - RequestorLabel
  - ManagedByLabel
  - ManagedBySuffix
  - **note** QuotaLabel is replaced, will be deprecated in v1alpha2 and removed in v1alpha3

Additionally, some new implementations require additional config. Which is documented in the rest of the 
Administrators guide, but documented here as well, so that Administrators converting from v1alpha1 to v1alpha2
also add these changes as part of the PaasConfig migration process.

## Conversion

The Paas operator can work with both v1alpha1 and v1alpha2.
Internally v1alpha1.PaasConfig is converted and stored as v1alpha2.PaasConfig, and converted back if the client requests
a v1alpha1.PaasConfig. Switching to v1alpha2.PaasConfig is therefore recommended, but can be separately from deploying v2.

## Changing v1alpha1 to v1alpha2

### Removing deprecated fields

The following fields in PaasConfig.Spec are removed in v1alpha2 and should be removed from the PaasConfig:
- argoenabled
- argopermissions
- exclude_appset_name
- groupsynclist
- groupsynclist_key
- ldap

### Adding custom fields with validations

With v2, the previous implementation (hardcoded fields in all capabilities, just required by ArgoCD) is removed.
There is now only [Custom fields](./capabilities.md#configuring-custom-fields).
We advise to add the following custom fields to your argocd capability:

!!! example

    ```
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: learn-paas-config
    spec:
      capabilities:
        argocd:
          custom_fields:
            git_url:
              required: true
            git_revision:
              default: "main"
            git_path:
              default: "."
    ```

### Labels and Capability fields

- [Custom fields for all capabilities](./go-templating.md#custom-fields-for-all-capabilities)
- [Labels with go templating](./go-templating.md#labels-with-go-templating) (v3 only)