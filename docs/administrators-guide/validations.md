---
title: Validating field names
summary: Options to configure validation of field names
authors:
  - devotional-phoenix
date: 2025-02-27
---

# Introduction

Most fields in the CRD are checked directly with kubebuilder validations,
But some fields can be validated with a regular expression that is configurable through the PaasConfig:

Below snippet shows how validations can be configured for the complete set of vailable validations:

!!! example

    ```yml
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      validations:
        paas:
          # Validate name of Paas
          name: "^[a-z0-9-]*$"
          # Validate name of groups in paas
          groupName: "^[a-z0-9-]*$"
          # Validate name of namespaces in paas
          namespaceName: "^[a-z0-9-]*$"
          # Validate requestor field in paas
          requestor: "^[a-z0-9-]*$"
        paasConfig:
          # Validate name of capability in config
          capabilityName: "^[a-z0-9-]*$"
        paasNs:
          # Validate name of paasNs
          name: "^[a-z0-9-]*$"
    ...
    ```

!!! note

    If only one of `PaasConfig.spec.validations.paas.namespaceName`, and `PaasConfig.validations.paasNs.name` is set,
    both PaasNs names and Paas.Spec.Namespaces are validated with the same validation rule.
