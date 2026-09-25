---
title: Configuring in the PaasConfig
summary: A detailed description of capabilities what it can do and how you can configure them in the PaasConfig CRD.
authors:
  - Devotional Phoenix
date: 2024-12-09
---

# Configuring in the PaasConfig

The Paas Operator can deliver capabilities to enable Paas deployments with CI and CD options with a one-click option.
Some examples of capabilities include:

- enabling ArgoCD for Continuous Delivery on your Paas namespaces
- enabling Tekton for Continuous Integration of your application components
- observing your Paas resources with Grafana
- configuring federated Authentication and Authorization with keycloak

Configuring capabilities does not require code changes / building new images. It only requires:

1. [configuration for the Paas operator](config/) via `PaasConfig`
2. an [ApplicationSet](applicationset) in the namespace of the cluster-wide ArgoCD
3. a git repository for the cluster-wide ArgoCD to be used for deploying the capability for a Paas with the capability defined

## 📘 Contents

- [Application Set](applicationset/)
  An example ApplicationSet to be defined for a capability

- [PaasConfig configuration for a capability](config/)
  What to define in the Paas Config for a single capability

- [Custom Fields](custom-fields/)
  How to define custom fields

- [Service Account Permissions](permissions/)
  How to define permissions for Service Accounts for a Capability

- [Capability Qupta management](quota/)
  - [Cluster‑Wide Quotas](cluster-wide-quotas/)
    Dynamic quota management with one quota pool for the entire capability for all paas'es

- [External capabilities](external-capabilities/)
  What is an External Capability
