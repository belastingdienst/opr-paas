---
title: Validating field names
summary: Options to configure validation of field names
authors:
  - devotional-phoenix
date: 2025-02-27
---

# Introduction

With release v1.5.0 we have added webhook validations, verifying groupnames against RFC 1035.
With release v1.5.1, we have expanded the group names validation to be configurable with a regular expression.
We might expand this feature to configure validation on other fields in future releases.

## Example PaasConfig

Below snippet shows how to set a regular expression on group names in a paas:

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      validations:
        paas:
          groupNames: "^[a-z0-9-]*$"
    ...
    ```
