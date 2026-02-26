---
title: Administrator's Guide
summary: The section for administrators setting up and maintaining the Paas Operator.
authors:
  - hikarukin
date: 2025-06-23
---

# Administrator’s Guide

Welcome to the Administrator’s Guide. This guide is intended for administrators
and operators responsible for deploying, configuring, securing, and maintaining
the Paas Operator in production environments.

---

## 📘 Contents

- [Installation](install/)  
  Step-by-step instructions to deploy the operator in your cluster.

- [Configuration](configuration/)  
  Guidance on customizing system behavior via `PaasConfig`.

- [Cluster‑Wide Quotas](cluster-wide-quotas/)  
  Instructions for enforcing resource usage limits across namespaces.

- [Capabilities](capabilities/)  
  Modular, plugin‑style features like ArgoCD, Tekton, Grafana, and Keycloak.

- [Secrets](secrets/)  
  Secure management of secrets within the operator.

- [Security](security/)  
  Best practices and hardening guidelines for production deployments.

- [Validations](validations/)  
  Built‑in checks to ensure correct configurations and prevent misconfigurations.

_For development workflows, release procedures, and contributor guidelines, see the [Developer’s Guide](../development-guide/index.md)._

---

## Version Support

We follow a **roll‑forward support model**. Only the **latest major version** is
supported. Previous major versions are considered **end-of-life (EOL)** and do not
receive updates, security patches, or fixes.

Administrators are expected to upgrade to the latest available version to remain supported.

For detailed information about our support policy, including versioning, hotfixes,
and the no‑backport rule, please refer to the [Support Policy in the Developer’s Guide](../development-guide/25_support-policy.md).
