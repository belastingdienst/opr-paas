---
title: Security notes
summary: A set of security related notes and tips with regards to running the PaaS operator.
authors:
  - hikarukin
date: 2024-08-21
---

# Introduction

For any piece of software, security is of paramount concern. With the PaaS operator,
we aim to provide safe, secure and sane defaults for our settings. If you have any
improvements you'd like to share, feel free to create an issue or a pull request (PR)
in our source code repository.

For more information on contributing to this project, please see the
[development guide](/development-guide) section in this documentation and the
CONTRIBUTING.md file in the root of our source code repository.

Should you find a security issue, please refer to the SECURITY.md file in the
root of our source code repository.

## Things to be aware of

### Automount is set to true

The operator makes use of a service account token that is used to communicate
with the Kubernetes APIs. This service account token is automatically mounted
using K8S's automount feature.

It is a common best-practice for normal pods to opt-out of automatically mounting
a service account token using `automountServiceAccountToken: false`.

However, since this concerns an operator that needs the service account for most
things it does, we have opted to keep the token auto-mounted.