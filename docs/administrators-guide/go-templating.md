---
title: Go templating options
summary: A detailed description of how we use go templating to change hardcoded options to PaasConfig
authors:
  - Devotional Phoenix
date: 2025-06-18
---

# Go templating generics

## Templating

The templating feature allows administrators to dynamically generate values from information in Paas and/or PaasConfig.
This provides flexibility for other organisations using the Paas operator with other business logic.

## Syntax

The template options support Go templating syntax, in which all values from the Paas and PaasConfig can be referenced, more examples below.
In addition to the default Go template functions, we've added support for
[all Sprout](https://docs.atom.codes/sprout/groups/all) Go template functions.

## Behavior of multi-valued and single valued results

Depending on the result of the go template, one of three things can happen:

- if the result can be parsed as list:
  - The key of the resulting item (label or custom field) is derived from the name of the template, suffixed with an underscore and an integer (number in the list of this item).
  - The value of the resulting item is the direct value of the item in the list
- if the result can be parsed as map:
  - The key of the resulting item (label or custom field) is derived from the name of the template, suffixed with an underscore and the key of the map item
  - The value of the resulting item is the direct value of the map item
- If the result is not parsable as list or map:
  - The key of the resulting item (label or custom field) is derived from the name of the template
  - The value of the resulting item is derived from the exact returned string

!!! Note
    If you want to return a map or list as a single value in a field you have the following options:
    - convert the map to a string representation using toYaml or toJson, and add quoting to make sure it is parsed as one string 
    - create a map with one key/value pair and set the resulting string as the value

## Developing go templates

For easier validation and debugging of templates, we recommend using [Repeat It](https://repeatit.io/), an online tool to test and validate your Go templates.

# Implementations

## Labels with go templating

Administrators can define labels to be added to resources managed by a Paas.
The implementation is based on go-templating, and has the Paas and Resource as inputs.
This feature can be used to:
- copy labels (or annotations) from the Paas, (or PaasConfig) to labels on the specific resource
- use specific fields in the Paas (or PaasConfig) to define extra labels

!!!! Note
     This feature replaces certain hardcoded implementations that where previously implemented.
     If you want to keep the behavior, please add the folling to your PaasConfig:

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      ...
      templating:
        clusterQuotaLabels:
          "": '{{ range $key, $value := .Paas.Labels }}{{ if ne $key "app.kubernetes.io/instance" }}{{$key}}: {{$value}}\n{{end}}{{end}}'
        groupLabels:
          "": '{{ range $key, $value := .Paas.Labels }}{{ if ne $key "app.kubernetes.io/instance" }}{{$key}}: {{$value}}\n{{end}}{{end}}'
        namespaceLabels:
          "": '{{ range $key, $value := .Paas.Labels }}{{ if ne $key "app.kubernetes.io/instance" }}{{$key}}: {{$value}}\n{{end}}{{end}}'
					"argocd.argoproj.io/managed-by": "{{ .Paas.Spec.ManagedByPaas }}-argocd"
        roleBindingLabels:
          "": '{{ range $key, $value := .Paas.Labels }}{{ if ne $key "app.kubernetes.io/instance" }}{{$key}}: {{$value}}\n{{end}}{{end}}'
    ```

## Capability fields with go templating

### Custom fields per capability

The Paas operator allows administrator to define custom fields which can be set by requestors and end up as fields in the list generator 
in the ApplicationSet that defines the Application for the capability for the Paas.

For more info, see [api-guide on capability custom field configuration in the Paas](../administrators-guide/capabilities.md#configuring-custom-fields)

### Custom fields for all capabilities

Addiotnally to setting custom fields for a specific capability, the Paas operator also allows administrator to define custom fields for all capabilities.
There are 2 main differences:
1. These custom fields cannot be overruled by a custom field for a specific Paas
2. These custom fields are generically applied to all capabilities.

!!!! Note
     This feature replaces certain hardcoded implementations that where previously implemented.
     If you want to keep the behavior, please add the following to your PaasConfig:

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      ...
      templating:
        genericCapabilityFields:
					requestor: "{{ .Paas.Spec.Requestor }}",
					service: "{{ (splitn \"-\" 2 .Paas.Name)._0 }}",
					subservice: "{{ (splitn \"-\" 2 .Paas.Name)._1 }}",
    ```

# Examples

This chapter shows some examples of go-templates that are used in this project and explains how they work and why they are phrased as such.

## RBAC block

The following example loops through the groups in the paas spec and generates a RBAC line for every group

!!! example

    ```jinja
    g, system:cluster-admins, role:admin{{ range $groupName, $group := .Paas.Spec.Groups }}
    g, {{ $groupName }}, role:admin{{end}}
    ...
    ```

## Reference PaasConfig values

You can reference values from the PaasConfig as well by referencing `.Config`:

!!! example

    ```jinja
    {{ .Config.Spec.Debug }}
    ```

## return multiple keys as a map

Your template could return a map (using `key: value` formatting) to return multiple key/value pairs

!!! example

    ```jinja
    debug: {{ .Config.Spec.Debug }}
    argo: {{ .Config.Spec.ArgoEnabled }}
    ```

This would return 2 key/value pairs. If name of the template would be set to `my_map`, values would have keys `my_map_debug` and `my_map_argo`.

## return multiple keys as a list

Your template could return a list (using `- value` formatting) to return multiple key/value pairs.

!!! example

    ```jinja
    - {{ .Config.Spec.Debug }}
    - {{ .Config.Spec.ArgoEnabled }}
    - custom fields with templating is cool
    ```

This would return 3 key/value pairs. If name of the template would be set to `my_list`, values would have keys `my_list_0` and `my_list_1`.

## Adding all labels, except for a specific key

Ideally this could be done using the [omit dict function](https://masterminds.github.io/sprig/dicts.html), but unfortunately, 
the dict is implemented as map[string]any, and labels are implemented as `map[string]string` and go does not automatically convert.

We have used a range and if statement to create all key/value pairs one by one.
Note that the go-template is spread across multiple lines.
This make sure every key is on a separate line, and thus parsed as separate key/value pair too.

!!! example

    ```jinja
    {{ range $key, $value := .Paas.Labels }}{{ if ne $key "app.kubernetes.io/instance" }}{{$key}}: {{$value}}
    {{end}}{{end}}
    ```