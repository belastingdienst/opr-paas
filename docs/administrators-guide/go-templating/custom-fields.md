---
title: Capability fields with Go Template
summary: Defining capability fields with Go Templating
authors:
  - Devotional Phoenix
date: 2026-09-21
---

# Capability fields with Go Template

## Custom fields per capability

The Paas operator allows administrator to define custom fields which can be set by requestors and end up as fields in the list generator
in the ApplicationSet that defines the Application for the capability for the Paas.

## Custom fields for all capabilities

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

