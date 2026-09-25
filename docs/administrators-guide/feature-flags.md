---
title: Configuring feature flags
summary: A detailed description of configuring features.
authors:
  - Devotional Phoenix
date: 2025-07-21
---

# Configuring features

To offer a configurable path to introduce new features, deprecate obsolete features and fine tune some implemented features,
the Paas operator offers feature flags.

## Warn or block groups with user management

One of the implemented Feature Flags is for the behavior when users have defined usernames in the Paas.Spec.Groups blocks.

### Allow (default)

When specifying `allow` (or leave empty), the operator reports no errors / warnings.

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      feature_flags:
        group_user_management: allow
    ```

### Warn

The option `warn` can be used to have the WebHook warn about users being set, without declining the request,
and have the controller log warnings to console and the Paas Status block.

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      feature_flags:
        group_user_management: warn
    ```

### Block

The option `block` can be set to decline requests with users being set in the Groups block,
have the controller log warnings to console and the Paas Status block, 
and have the controller remove groups that have previously been defined.

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      feature_flags:
        group_user_management: block
    ```

## Warn or block cluster resource quota management

This flag controls whether the operator manages ClusterResourceQuotas for a Paas
and its capabilities, as defined in the Paas.Spec.Quota and Paas.Spec.Capabilities[].Quota blocks.

### Allow (default)

When specifying `allow` (or leave empty), the operator creates and manages ClusterResourceQuota's as configured.

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      feature_flags:
        cluster_resource_quota_management: allow
    ```

### Warn

The option `warn` can be used to have the WebHook warn about quotas being set, without declining the request,
and have the controller ignore the specified quotas.

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      feature_flags:
        cluster_resource_quota_management: warn
    ```

### Block

The option `block` can be set to decline requests with quotas being set in the Quota blocks,
and have the controller remove ClusterResourceQuota's that have previously been created.

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      feature_flags:
        cluster_resource_quota_management: block
    ```
