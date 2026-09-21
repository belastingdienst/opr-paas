---
title: Go Template Examples
summary: Some examples for Go Template configuration
authors:
  - Devotional Phoenix
date: 2025-06-18
---

# Examples

This chapter shows some examples of Go Templates that are used in this project and explains how they work and why they are phrased as such.

## Reference PaasConfig values

You can reference values from the PaasConfig by using `.Config`:

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

## Adding multiple key/values, except for a specific key

Ideally this could be done using the [omit dict function](https://masterminds.github.io/sprig/dicts.html), but unfortunately,
the dict is implemented as map[string]any, and labels are implemented as `map[string]string` and go does not automatically convert.

Therefore, we have used a range and if statement to create all key/value pairs one by one.

!!! note
    The Go Template is spread across multiple lines.
    This ensures that each key is placed on a separate line, and is thus correctly parsed as an individual key/value pair.

!!! example

    ```jinja
    {{ range $key, $value := .Paas.Labels }}{{ if ne $key "app.kubernetes.io/instance" }}{{$key}}: {{$value}}
    {{end}}{{end}}
    ```

## RBAC block

The following example is used in the ArgoCD capability to generate and RBAC block

!!! example

    ```jinja
    g, system:cluster-admins, role:admin{{ range $groupName, $group := .Paas.Spec.Groups }}
    g, {{ $groupName }}, role:admin{{end}}
    ...
    ```
