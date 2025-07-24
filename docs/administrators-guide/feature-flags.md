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

Currently the only implemented Feature Flag is for the behavior when users have defined usernames in the Paas.Spec.Groups blocks.

### Allow (default)

When specifying `allow` (or leave empty), the operator reports no errors / warnings.

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha1
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