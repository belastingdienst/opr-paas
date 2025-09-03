---
title: Options to have additional logging
summary: A detailed description of how debug logging can be configured.
authors:
  - Devotional Phoenix
date: 2025-09-02
---

# Introduction

By default the operator has a lot of logging up to the INFO level.
There are 2 options to add additional logging, and they can be set in 2 ways.
- enable all debug logging from commandline with the `-debug` option flag
- enable debug logging for one or more components from the commandline with the `-component-debug` option flag
- enable all debug logging with the PaasConfig with the `PaasConfig.spec.debug` option
- enable or disable debug logging for one or more components with the `PaasConfig.spec.components_debug` map

## Debug option

The operator can switch to debug mode, which enables debug logging for all components.
To enable the debug mode, run the operator with an additional commandline argument `-debug`,
or set `PaasConfig.spec.debug` to true.

!!! note
    Changing the commandline option changes the Deployment and restarts the operator in debug-mode.
    Changing `PaasConfig.spec.debug` to true does not trigger a restart, and enables debug-mode temporarily.
    Removing `PaasConfig.spec.debug` (or setting to false) switches back to normal operation.
    When commandline option `-debug` is used, changing `PaasConfig.spec.debug` has no effect.

## Component debugging

The operator has many (moving) parts and enabling debug-mode for everything creates a lot of logging, especially on 
clusters with many Paas'es. For this reason, we have developed an advanced debug feature called component logging.
With component logging, you have the option to enable debug-mode, with only for specific parts of the operator (which 
we call components).

We currently have implemented debug-mode for the following components:
- Webhooks:
  - v1alpha1:
    - paasconfig_webhook_v1
    - paas_webhook_v1
    - paasns_webhook_v1
    - utils_webhook_v1
  - v1alpha2:
    - paasconfig_webhook_v2
    - paas_webhook_v2
    - paasns_webhook_v2
    - utils_webhook_v2
- Controllers:
  - capabilities_controller
  - cluster_quota_controller
  - cluster_role_binding_controller
  - group_controller
  - namespace_controller
  - paas_controller
  - paas_config_controller
  - rolebinding_controller
  - secret_controller
- plugin_generator
- config_watcher

To enable component debugging from the commandine, you can use the `-component-debug` commandline argument and supply
all components as one comma separated string 

!!! example
    -component-debug secret_controller,plugin_generator,paasconfig_webhook_v1

Alternately you can enable or disable component debugging `PaasConfig.spec.components_debug` by specifying the name of 
the component as key and either `true` or `false` for either `on` or `of` respectively.

!!! example
    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      components_debug:
        paas_controller: true
        secret_controller: false
    ```

## Precedence
Precedence is as follows:
- if `PaasConfig.spec.components_debug` is set, that value is used.
- if the commandline argument for `-debug` is used, and/or `PaasConfig.spec.debug` is set to true, all components
  are in debug-mode (except for those set to `false` in `PaasConfig.spec.components_debug`).
- if `-debug` is not used, and `PaasConfig.spec.debug` is not set to false, and a component is not present in 
  `PaasConfig.spec.components_debug`, but is added in `-component-debug` the component will also run in debug-mode.

An example of how you could utilize these options:
- For auditing reasons you might want to have more info on v1alpha1 requests. Therefore you set the following 
  commandline argument: `-component-debug paasconfig_webhook_v1,paas_webhook_v1,paasns_webhook_v1,utils_webhook_v1`.
- Then something unexpected happens. To investigate the issue, you enable debug-mode for all components by setting:
  `PaasConfig.spec.debug=true`.
- You soon find out the specific components you want to further investigate. You disable `PaasConfig.spec.debug=true`
  and enable the specific components in `PaasConfig.spec.components_debug`. Also you want to temporarily disable the 
  webhook logging that was switched on by the commandline arguments. You set:
  ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      components_debug:
        paas_controller: true
        secret_controller: true
        paasconfig_webhook_v1: false
        paas_webhook_v1: false
        paasns_webhook_v1: false
        utils_webhook_v1: false
  ```
- The issues is resolved and to swotch back to normal operation you remove the `PaasConfig.spec.components_debug`.
  - all v1alpha1 webhooks are back in ebug-mode
  - all other components are not in debug-mode anymore