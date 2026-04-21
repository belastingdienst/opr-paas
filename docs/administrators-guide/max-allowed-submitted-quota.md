---
title: Max Allowed Submitted Quota
summary: Enforcing upper bounds on user-submitted quota requests.
authors:
  - hikarukin
date: 2026-03-10
---

# Max Allowed Submitted Quota (v1alpha2)

The `MaxAllowedSubmittedQuota` feature allows administrators to define a global "ceiling" for quotas requested by users in a `Paas` resource.

While `PaasConfig.spec.capabilities` defines default and per-capability quotas, `MaxAllowedSubmittedQuota` acts as a final validation guardrail at the `Paas` level.

## Configuration

This is configured in the `PaasConfig` (v1alpha2) under `.spec.maxAllowedSubmittedQuota.maxQuota`.

!!! example "PaasConfig Snippet"

    ```yaml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      maxAllowedSubmittedQuota:
        maxQuota:
          limits.cpu: "8"
          limits.memory: 16Gi
          requests.cpu: "4"
          requests.memory: 8Gi
          requests.storage: "100Gi"
    ```

## How it works

1. **Admission Control**: When a user creates or updates a `Paas` resource, the validating webhook compares the values in `Paas.spec.quota` against the values defined in `PaasConfig.spec.maxAllowedSubmittedQuota.maxQuota`.
2. **Quantity Comparison**: Comparison is performed using Kubernetes `resource.Quantity` logic (e.g., `1000m` is equal to `1`).
3. **Denial**: If any requested resource exceeds the maximum allowed value, the request is rejected with an error message indicating which resource violated the policy.

!!! failure "Example Denial"
    If the max allowed `limits.cpu` is `8` and a user submits a `Paas` with `limits.cpu: "10"`, the webhook returns:
    `quota (limits.cpu) cannot be larger than MaxAllowedSubmittedQuota (8)`

## Key Validations

The keys used in `maxQuota` (e.g., `limits.cpu`) are subject to the same regex validation as other quota fields in the operator. 

If you have configured `PaasConfig.spec.validations.paas.allowedQuotas`, any key added to `MaxAllowedSubmittedQuota` must match that regular expression. If it does not, the `PaasConfig` itself will be rejected during creation or update.

For more details on regex validation, see the [Validations guide](validations.md).

## Important Notes

- **Empty Quotas**: If a `Paas` does not define any quotas in `.spec.quota`, this validation is skipped.
- **Guardrail Only**: This feature is a validation guardrail for the `Paas` custom resource. It does not replace or modify standard Kubernetes `ResourceQuota` or `LimitRange` objects in the underlying namespaces.