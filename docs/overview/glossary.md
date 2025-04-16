---
title: Glossary
summary: A short glossary of terms in relation to the Paas operator.
authors:
  - hikarukin
date: 2024-07-09
---

Glossary
========

## Capabilities

A capability is extra functionality that can be added to your Paas simply by
enabling it through your PaasConfig.

Examples can include, but are not limited to, ArgoCD, Tekton or Grafana.

## Cluster Wide Quotas

With Cluster Wide (resource) Quotas (CWQ), cluster admins can bring all resources
for all Paas'es belonging to a capability, together in one cluster wide resource pool.

This brings down over commit at the expense of the risk of resource sharing.

For more details, see the [relevant details in the administrators section](administrators-guide/cluster-wide-quotas/basic-usage.md)

## Crypttool

The crypttool is a small command-line utility that allows a user to perform some
simple operations in regard to secrets in a Paas.

Basic functionality includes sub-commands for `encrypt`, `decrypt` and `re-encrypt`
in regard to Paas related secrets.

The `re-encrypt` sub-command specifically will parse the yaml/json file for a Paas,
decrypt the SSH secrets with the previous private key, re-encrypt with the new public
key & write back the Paas configuration to the file in either yaml or json format.

This will allow for key rotation.

It can also be used to `generate` a new public/private key pair.

The crypttool is managed from its own repository on GitHub at [https://github.com/belastingdienst/opr-paas-crypttool](https://github.com/belastingdienst/opr-paas-crypttool).

## Groups [openshift]

Access to a Paas is granted to specific groups, which can be listed in the PaasConfig.

A group can contain roles that allow them certain permissions, users and/or an
LDAP query. When configured, the LDAP query is used to find the members of the
group, *in addition* to any users listed specifically in the PaasConfig.

Please be aware that this is an OpenShift specific feature.

## ManagedByPaas

This is a field in the PaasConfig, and feature, that allows the user to
indicate that this Paas is actually managed by another Paas' ArgoCD.

## Namespace / PaasNs

Namespaces can be used to define extra namespaces to be created as part of this
Paas project.

## Quotas

There a various quotas that can be configured, but essentially they are: cluster
wide, per Paas or for a capability.

## Requestor

The requestor is an informational field in the PaasConfig, which can contain
a string that is intended to point to the person or group responsible for the
application / Paas.

This could be an ITIL group, Service desk name, email address or any random string.

## SSH Secrets

You can add SSH keys, which are a type of secret, to your Paas for ArgoCD to use
so it can access a git repository. For example on a self-hosted Github or BitBucket
instance.

The SSH secrets must be encrypted with the public key corresponding to the
private key that was deployed together with the Paas operator.

## Web service

The web service exposes a `/v1/encrypt` endpoint that allows a user to encrypt a
secret using that cluster's private key.

Apart from the encrypt endpoint, some standard endpoints like `/healthz`, `/readyz`,
`/version` and `/metrics` are exposed.
