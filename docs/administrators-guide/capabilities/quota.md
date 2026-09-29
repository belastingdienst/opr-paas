---
title: Configuring Default Quota
summary: A detailed description of capabilities what it can do and how you can configure them in the PaasConfig CRD.
authors:
  - Devotional Phoenix
date: 2024-12-09
---

# Configuring Default Quota

## Introduction

For every Capability for every Paas, a separate ClusterResourceQuota is created.
Quotas can be set in a Paas, and when not set, the Capability configuration can have a Default which will be used instead.
Furthermore, the capability configuration can also have a min and max value set.
The Paas operator will use the value as set in the Paas, and these Default, Min and Max settings to come to the proper value to be set in the ClusterResourceQuota set on the namespace.
Beyond these options, a capability can also be configured to use cluster-wide Quota with the `spec.capabilities["new-capability"].quotas.clusterwide`
and `spec.capabilities["new-capability"].quotas.raio`.

## More info

For more information please check:

- [administrators-guide's Cluster Wide Quotas section](./cluster-wide-quotas/)
- [api-guide on capability quota configuration](../../development-guide/00_api.md#configcapability)
- [api-guide on capability quota in the Paas](../../development-guide/00_api.md#paascapability)
