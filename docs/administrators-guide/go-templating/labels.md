---
title: Go Template Label options
summary: A detailed description of how Go Template is used to define labels
authors:
  - Devotional Phoenix
date: 2025-06-18
---

# Labels with go templating

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
