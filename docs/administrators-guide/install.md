---
title: Installing the Operator
summary: A simple guide on installing the Paas Operator
authors:
  - hikarukin
date: 2024-10-14
---

# Introduction

Deploy the operator using the following command:

```
kubectl apply -f https://github.com/belastingdienst/opr-paas/releases/latest/download/install.yaml
```

This will install the operator using the install.yaml that was generated for the
latest release. It will create:

- a namespace called `paas`;
- 2 CRDs (`Paas` and `PaasNs`);
- a service account, role, role binding, cluster role & cluster role binding for
  all permissions required by the operator; **In this install.yaml, the serviceaccount gets the: `bind` permission to bind any role.
It is advised to follow the principle of least privileged and set these privilege to only allow binding of the roles set in your
operator config.**
- a configmap with an example configuration for the operator;
- a secret with an example key for sshDecrypt **CHANGE THESE IN PRODUCTION**
- a deployment running the operator;

Feel free to change config as required.