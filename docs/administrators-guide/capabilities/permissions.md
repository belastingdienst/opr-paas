---
title: Configuring Service Account Permissions
summary: A detailed description of capabilities what it can do and how you can configure them in the PaasConfig CRD.
authors:
  - Devotional Phoenix
date: 2024-12-09
---

# Configuring Service Account Permissions

## Introduction

For every capability the Paas operator can grant permissions to service accounts.
There are two options:

- Default permissions: These permissions are granted for this capability for every Paas
- Extra permissions: These permissions are granted only when a Paas has set `spec.capabilities[capability].extra_permissions` to true
  The main goal for extra permissions is to start off with higher permissions to get started, and revert them when a lower permissive option is available (e.a. lower permissions are set as default permissions).
  Customers starting with extra permissions can test with default permissions and return to extra permissions if they run into issues.

## More info

For more information on Default permissions and Extra permissions please refer to:

- [Example PaasConfig with a capability](config/)
- [api-guide on capability configuration in the PaasConfig](../../development-guide/00_api.md#configcapability)
- [api-guide on capability configuration in the Paas](../../development-guide/00_api.md#paascapability)
