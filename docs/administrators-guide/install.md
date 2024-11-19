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

This will install the operator using the `install.yaml` that was generated for the
latest release. It will create:

- a namespace called `paas-system`;
- 2 CRDs (`Paas` and `PaasNs`);
- a service account, role, role binding, cluster role & cluster role binding for
  all permissions required by the operator; **As the operator binds role for others the serviceaccount gets the: `bind` permission.
  It is advised to follow the principle of least privilege and scope the `permission` to only allow binding of the roles set in your
  operator config by setting `resourcesNames` in your role.yaml**
- a viewer & an editor cluster role for Paas and PaasNs resources;
- a configmap with all operator configuration options;
- a deployment running the operator;

Feel free to change config as required.