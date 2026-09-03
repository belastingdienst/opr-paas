---
title: Go Template options
summary: A detailed description of how Go Template is used to replace hardcoded options with dynamic PaasConfig values.
authors:
  - Devotional Phoenix
date: 2025-06-18
---

# Go Template

## Template

The template feature allows administrators to dynamically generate values from information in Paas and/or PaasConfig.
This provides flexibility for other organizations using the Paas operator with other business logic.

## Syntax

Template options support standard Go Template syntax, allowing all values from the Paas and PaasConfig to be referenced. See more examples below.
In addition to the default Go Template functions, we've added support for
[all Sprout](https://docs.atom.codes/sprout/registries/list-of-all-registries) Go Template functions,
and [backward](https://docs.atom.codes/sprout/registries/backward) (we want to use the `fail` function.

## Behavior of multivalued and single valued results

Depending on the result of the Go Template, one of three things can happen:

- if the result can be parsed as a list:
  - The key of the resulting item (label or custom field) is derived from the name of the template, suffixed with an underscore and an integer (number in the list of this item).
  - The value of the resulting item is the direct value of the item in the list
- if the result can be parsed as map:
  - The key of the resulting item (label or custom field) is derived from the name of the template, suffixed with an underscore and the key of the map item
  - The value of the resulting item is the direct value of the map item
- If the result is not parsable as list or map:
  - The key of the resulting item (label or custom field) is derived from the name of the template
  - The value of the resulting item is derived from the exact returned string

!!! note

    If you need to return a map or list as a single string value in a field, you have the following options:
    - convert the map to a string representation using toYaml or toJson, and add quoting to make sure it is parsed as one string
    - create a map with one key/value pair and set the resulting string as the value

## Developing Go Templates

For easier validation and debugging of templates, we recommend using [Repeat It](https://repeatit.io/), an online tool to test and validate your Go Templates.

# Implementations

## Labels with go templating

Administrators can define labels to be added to resources managed by a Paas.
The implementation is based on Go Templating, and has the Paas and Resource as inputs.
This feature can be used to:

- copy labels (or annotations) from the Paas, (or PaasConfig) to labels on the specific resource
- use specific fields in the Paas (or PaasConfig) to define extra labels

!!! note

    This feature replaces certain hardcoded implementations that were previously implemented.
    If you want to keep the behavior, please add the following to your PaasConfig:

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
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
          "argocd.argoproj.io/managed-by": "{{ .Paas.Spec.ManagedByPaas | default .Paas.Name }}-argocd"
        roleBindingLabels:
          "": '{{ range $key, $value := .Paas.Labels }}{{ if ne $key "app.kubernetes.io/instance" }}{{$key}}: {{$value}}\n{{end}}{{end}}'
    ```

## Capability fields with Go Template

### Custom fields per capability

The Paas operator allows administrator to define custom fields which can be set by requestors and end up as fields in the list generator
in the ApplicationSet that defines the Application for the capability for the Paas.

For more info, see [api-guide on capability custom field configuration in the Paas](../administrators-guide/capabilities.md#configuring-custom-fields)

### Custom fields for all capabilities

In addition to setting custom fields for specific capabilities, the Paas operator also allows administrators to define custom fields that apply to all capabilities.
There are two main differences:
1. These custom fields cannot be overruled by a custom field for a specific Paas
2. These custom fields are generically applied to all capabilities.

!!! note

    This feature replaces certain hardcoded implementations that were previously implemented.
    If you want to keep the behavior, please add the following to your PaasConfig:

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
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

This chapter shows some examples of Go Templates that are used in this project and explains how they work and why they are phrased as such.

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

## Return multiple keys as a map

Your template could return a map (using `key: value` formatting) to return multiple key/value pairs

!!! example

    ```jinja
    debug: {{ .Config.Spec.Debug }}
    argo: {{ .Config.Spec.ArgoEnabled }}
    ```

This would return two key/value pairs. If name of the template would be set to `my_map`, values would have keys `my_map_debug` and `my_map_argo`.

## Return multiple keys as a list

Your template could return a list (using `- value` formatting) to return multiple key/value pairs.

!!! example

    ```jinja
    - {{ .Config.Spec.Debug }}
    - {{ .Config.Spec.ArgoEnabled }}
    - custom fields with templating is cool
    ```

This would return three key/value pairs. If name of the template would be set to `my_list`, values would have keys `my_list_0` and `my_list_1`.

## Adding all labels, except for a specific key

Ideally this could be done using the [omit dict function](https://masterminds.github.io/sprig/dicts.html), but unfortunately,
the dict is implemented as map[string]any, and labels are implemented as `map[string]string` and go does not automatically convert.

We have used a range and if statement to create all key/value pairs one by one.
Note that the Go Template is spread across multiple lines.
This ensures that each key is placed on a separate line, and is thus correctly parsed as an individual key/value pair.

!!! example

    ```jinja
    {{ range $key, $value := .Paas.Labels }}{{ if ne $key "app.kubernetes.io/instance" }}{{$key}}: {{$value}}
    {{end}}{{end}}
    ```

## Secrets with go-templating

The original secret implementation was specifically designed for generating ArgoCD repo secrets, but the secrets were less useful for other capabilities, namespaces, etc.
See [issue 1056](https://github.com/belastingdienst/opr-paas/issues/1056) for more info.

We have now replaced the implementation with a go-templating driven alternative, with allows administrators to define the exact format for secrets per capability, as well as for all namespaces specifically.
In the future we expect to expand the solution to
- also support External Secrets integration
- replace the default template (which currently defaults to the argocd behavior which has very limitted use outside of argocd

### How it works

In original solution a specific secret format was decided and applied for all secrets created by the operator.
In the new solution, Administrators can define go templates in the PaasConfig to define the format of secrets.
The template can be set in PaasConfig.spec.namespace_secrets, which will define the format for all secrets in namespaces defined in a Paas.spec.namespaces, and/or as a PaasNs.
Templates can also be set in PaasConfig.spec.capabilities[].secrets, in which case it will determine the format for a specific capability.

### Format

The templates can use info from the PaasConfig, and from the Paas. They can use special functions (described in [below chapter Special functions](#Special_functions)).
The template should result in a yaml string containing a map of maps (map[string]map[string]string{}).
In this result, the key for the first level defines the name of the secret to be created. The key for the second level defines the key in .data to be set to the value.

So, if the template resolves into
!!! example
    ```yaml
    my-secret: 
      my-key-1: some secret value
      my-key-2: some other secret value
    my-other-secret:
      other-key: other secret
    ```

The paas operator would create 2 secrets:
- one called `my-secret`, having 2 key/value pairs:
  - `my-key-1` which is set to the base64 version of the string `some secret value`
  - `my-key-2`, which is set to the base64 version of the string `some other secret value`
- another one called `my-other-secret` having only one pair:
  - `other key`, which is set to the base64 version of the string `other secret`

### If you don't set a template

For now we have added a default, which results in the same behavior as before this implementation was introduced:

!!! example
    ```
    {{- $secrets := dict -}}
    {{- range $key, $value := getPaasSecrets -}}
      {{- $hash := $key | sha512Sum | trunc 8 -}}
      {{- $secretName := print "paas-ssh-" $hash -}}
      {{- $secretData := dict "type" "git" "url" $key "sshPrivateKey" $value -}}
      {{- $_ := $secrets | set $secretName $secretData -}}
    {{- end -}}
    {{- $secrets | toYAML -}}
    ```
!!! note
    In the next major version we plan to require a template to be specified, and will then remove this default.

### Special functions

As you can see in the examples, we have added a function called getPaasSecrets.
This function parses all secrets that apply for this namespace, which could be
- For a capability: capability secrets and paas secrets
- For a non-capability namespace (Paas.spec.namespaces, or a PaasNs): namespace secrets and paas secrets.

Next to getPaasSecrets, we have also added getPaasSecret (without the s).
This function can be used to retrieve the (decrpyted) value of a specific secret.
Example usage: `{{ mySecret := getPaasSecret "my-secret" }}`

### Adviced template for ArgoCD

Below template can be used for the ArgoCD capability:

!!! example
    ```
    {{- $secrets := dict -}}
    {{- range $key, $value := getPaasSecrets -}}
      {{- $hash := $key | sha512Sum | trunc 8 -}}
      {{- $secretName := print "paas-ssh-" $hash -}}
      {{- $secretData := dict "type" "git" "url" $key "sshPrivateKey" $value -}}
      {{- $_ := $secrets | set $secretName $secretData -}}
    {{- end -}}
    {{- $secrets | toYAML -}}
    ```

This results in a secret for every paas secret, where the name of the paas secret is used as the url, and the value as the actual secret.
The name of the secret is derived from the hashed value of the name of the paas secret.

### Adviced template for Tekton

Below template can be used for the Tekton capability:

!!! example
    ```
    {{- $auths := dict -}}
    {{- range $name, $decrypted := getPaasSecrets -}}
      {{- $auth := base64Encode $decrypted -}}
      {{- $parts := split ":" $decrypted -}}
      {{- $username := index $parts "_0" -}}
      {{- $password := index $parts "_1" -}}
      {{- $authData := dict "username" $username "password" $password "auth" $auth -}}
      {{- $_ := $auths | set $name $authData -}}
    {{- end -}}
    {{- $dockerConfigJSON := toJSON (dict "auths" $auths) -}}
    {{- $secrets := dict "container-repo-token" (dict ".dockerconfigjson" $dockerConfigJSON) -}}
    {{ $secrets | toYAML }}
    ```

This results in one secret called container-repo-token, with one value called .dockerconfigjson.
All PaasSecrets are added as a auth token to the json value in `.dockerconfigjson`.

### Everything else

For all other situations you could fall back to this template:

!!! example
    ```
    {{- $scrt := getPaasSecrets -}}
    {{- if gt (len $scrt) 0 -}}
      {{- $result := dict "paas-secrets" $scrt -}}
      {{- $result | toYAML -}}
    {{- end -}}
    ```

This results in one secret, with all paas-secrets added as key/value pairs.
This will also be the future default when no template is set.

If you need another go-template, feel free to request support by issuing a ticket.
