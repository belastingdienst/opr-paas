---
title: Basic usage
summary: A basic usage description of how to use CWQs.
authors:
  - devotional-phoenix-97
  - hikarukin
date: 2024-07-01
---

Basic usage of CWQs
===================

With Cluster Wide Quotas, cluster admins can bring all resources for all Paas'es 
belonging to a capability together in one cluster wide resource pool. This brings
down over commit at the expense of the risks associated with resource sharing.

Use a quota per Paas
--------------------

Set:

- `paasconfig.spec.capabilities['tekton'].quotas.clusterwide` to `false`

Use CWQs with one hard-set value
--------------------------------

You can use CWQs with a single hard-set value (e.a. 10).

Set:

- `paasconfig.spec.capabilities['tekton'].quotas.clusterwide` to `true`
- `paasconfig.spec.capabilities['tekton'].quotas.ratio` to `0`
- `paasconfig.spec.capabilities['tekton'].quotas.min` to `10`

Use CWQs with autoscaling
-------------------------

You can use cluster wide quotas with an autoscaling feature.

For this example: every Paas is expected to use 1 CPU, and a minimum of 3 CPU
should always be available. Additionally, a maximum of 10 CPU can be reserved,
and we scale down to 10% of normal usage.

Set:

- `paasconfig.spec.capabilities['tekton'].quotas.clusterwide` to `true`
- `paasconfig.spec.capabilities['tekton'].quotas.default` to `1`
- `paasconfig.spec.capabilities['tekton'].quotas.min` to `3`
- `paasconfig.spec.capabilities['tekton'].quotas.max` to `10`
- `paasconfig.spec.capabilities['tekton'].quotas.ratio` to `0.1` (10%)
