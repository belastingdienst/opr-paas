---
title: Configuring capability Custom Fields
summary: A detailed description on how to configure Custom Fields for a capability
authors:
  - Devotional Phoenix
date: 2026-09-21
---

# Configuring capability Custom Fields

## Introduction

Capabilities might require options to be set in a Paas. The fields to be set would be specific to a capability.
Some examples include:

- setting a git url, revision and path for a ArgoCD bootstrap application
- setting a version for the Keycloak capability
- deploying multiple streams of a capability and allowing some DevOps teams to run a `latest` while others run a `stable` stream

For this reason we have introduced options for setting custom fields in the capability configuration in PaasConfig.
Each custom field belongs to a capability (e.a. `capability_name`), has a field name (e.a. `custom_field_name`) and configuration.
A custom field can be defined in PaasConfig in `PaasConfig.spec.capabilities[capability].customfields`.

The following configuration can be set:

- validation: A regular expression used to validate input in a Paas
- required: When set to true, an error is returned when the custom field is not defined in a Paas
- default: When set, a Paas without the custom field set will use this default instead.
- template: When set to a valid go template, the template is processed against the current Paas
  and PaasConfig end results are added as one or more custom fields in the ApplicationSet.
  (see [custom field templating](./go-templating/custom-fields/) for more information )

!!! note

    `required` and `default` are mutually exclusive.

When set, a Paas can set these custom_fields, which brings them to the generators field in the Application created by the ApplicationSet for this specific Paas.

## How a custom field operates

Imagine that on a cluster with

- a PaasConfig as defined in [Example PaasConfig with a capability](#example-paasconfig-with-a-capability), and
- an ApplicationSet as defined in [Example capability ApplicationSet](#example-ApplicationSet),
  a DevOps engineer created a Paas with a content like:

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: Paas
    metadata:
      name: my-paas
    spec:
      capabilities:
        mycap:
          custom_fields:
            my-custom-key: key_123
      ...
    ```

The following would happen:

- The Paas operator would check all custom_fields and use the following field values:
  - my-custom-key: key_123
  - my-custom-revision main
- The Paas operator would create an entry in the list generator in the ApplicationSet, with custom fields set as elements
- The cluster-wide ArgoCD ApplicationSet controller would create a new Application for my-paas-capability-mycap
- the Application would have the following set in `spec.source.kustomize.commonLabels`:
  - key: key_123
  - revision: main
- From here, Kustomize could use these values to be set on all resources create by the cluster-wide ArgoCD for this capability for this Paas

## Templating

The templating feature allows administrators to dynamically generate values for custom fields in the ApplicationSet without 
requiring users to explicitly specify these values in their Paas. This provides flexibility by enabling values to be derived from 
the Paas, the PaasConfig, or a combination of both. The template support Go templating syntax, in which all values from the Paas 
and PaasConfig can be referenced, more examples below. In addition to the default Go template functions, we've added support for
[all Sprout](https://docs.atom.codes/sprout/groups/all) Go template functions.

For more info on templating, see [custom fields templating](go-templating/custom-fields/)

### Precedence and overrides

- Paas values will take precedence over template values. If a custom field is defined in the Paas, its value will override the template.
- When a custom field is configured with a template, it will take precedence over other settings like default, validation, and required.
This means that the template value will override any default or validation settings configured for that field.

### Multi-value Fields

Templates return a string. If the operator can parse the returned value as YAML (or jinja) into a map or list, will result in a 
multi-value entry in the ApplicationSet. 
- For maps: The custom field name will be suffixed with the map keys
- For lists: The custom field name will be suffixed with the list index

### Examples

#### Referencing Paas values

You can reference values from the Paas by referencing `.Paas`:

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      clusterwide_argocd_namespace: paas-capabilities-argocd
      capabilities:
        mycap:
          ApplicationSet: mycap-as
          custom_fields:
            paas_name:
              template: "{{ .Paas.Name }}"
          quotas:
            defaults:
              limits.cpu: "8"
    ```

#### Referencing PaasConfig values

You can reference values from the PaasConfig as well:

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      clusterwide_argocd_namespace: paas-capabilities-argocd
      capabilities:
        mycap:
          ApplicationSet: mycap-as
          custom_fields:
            debug:
              template: |
                {{ .Config.Spec.Debug }}
            my-custom-revision:
              validation: '^(main|develop|feature-.*)$'
              default: main
          quotas:
            defaults:
              limits.cpu: "8"
    ```

#### loops

You can loop over lists and maps.
This example generates an argocd policy by ranging over the groups in the paas:

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      clusterwide_argocd_namespace: paas-capabilities-argocd
      capabilities:
        mycap:
          ApplicationSet: mycap-as
          custom_fields:
            argocd-policies:
              template: |
                g, system:cluster-admins, role:admin{{ range $groupName, $group := .Paas.Spec.Groups }}
                g, {{ $groupName }}, role:admin{{end}}
            my-custom-revision:
              validation: '^(main|develop|feature-.*)$'
              default: main
          quotas:
            defaults:
              limits.cpu: "8"
    ```

!!! note

    In the above example, you see the first line and the range on line 1, and the templated lines and end block on line 2.
    This causes that for every line a \n and after that a new row is inserted.
    This in turn leaves out the ending \n, which is unwanted.

    So, if you happen to see a |+ and extra \n in the resulting appset list generator value,
    this can be fixed by changing they way all is joined / seperated on lines in the template.

#### generating multiple custom fields with one template returning a map

You can return a map and create multiple keys (string suffix).

This would create 2 keys:

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      clusterwide_argocd_namespace: paas-capabilities-argocd
      capabilities:
        mycap:
          ApplicationSet: mycap-as
          custom_fields:
            "paas_config":
              template: |
                debug: {{ .Config.Spec.Debug }}
                argo_enabled: false
            my-custom-revision:
              validation: '^(main|develop|feature-.*)$'
              default: main
          quotas:
            defaults:
              limits.cpu: "8"
    ```

Which results in the following applicationSet entries:

!!! example

    ```yml
    apiVersion: argoproj.io/v1alpha1
    kind: ApplicationSet
    metadata:
      name: mycap-as
      namespace: paas-capabilities-argocd
    spec:
      generators:
        - list:
            elements:
              - paas_config_debug: true
                paas_config_argo_enabled: false
      ...
    ```

#### generating multiple custom fields with one template returning a list

You can also specify a list in the .Template spec and create multiple keys (number suffix).

This:

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      clusterwide_argocd_namespace: paas-capabilities-argocd
      capabilities:
        mycap:
          ApplicationSet: mycap-as
          custom_fields:
            "paas_config":
              template: |
                - {{ .Config.Spec.Debug }}
                - {{ .Config.Spec.ArgoEnabled }}
                - custom fields with templating is cool
            my-custom-revision:
              validation: '^(main|develop|feature-.*)$'
              default: main
          quotas:
            defaults:
              limits.cpu: "8"
    ```

Would create 3 ctsom fields:

!!! example

    ```yml
    apiVersion: argoproj.io/v1alpha1
    kind: ApplicationSet
    metadata:
      name: mycap-as
      namespace: paas-capabilities-argocd
    spec:
      generators:
        - list:
            elements:
              - paas_config_0: true
                paas_config_1: false
                paas_config_2: custom fields with templating is cool
      ...
    ```

## More info

For more information on Custom Fields please revert to:

- [Example PaasConfig with a capability](paasconfig/)
- [Example capability ApplicationSet with custom_fields being set as commonLabels](applicationset/)
- [api-guide on capability configuration in the PaasConfig](../development-guide/00_api.md#configcustomfield)
- [api-guide on capability configuration in the Paas](../development-guide/00_api.md#paascapability)
