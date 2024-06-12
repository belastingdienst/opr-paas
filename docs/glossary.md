---
title: Glossary
summary: A short glossary of terms in relation to the PAAS operator.
authors:
  - hikarukin
date: 2024-07-09
---

Glossary
========

## Capabilities

A capability is extra functionality that can be added to your PAAS simply by
enabling it through your PAAS' yaml config.

Examples can include, but are not limited to, ArgoCD, Tekton or Grafana.

## Cluster Wide Quotas

With cluster wide resource quotas, cluster admins can bring all resources for all
PAAS'es belonging to a capability, together in one cluster wide resource pool.

This brings down overcommit at the expense of the risk of resource sharing.

For more details, see the [relevant details in the administrators section](admin-guide/cwq/cluster-wide-quotas.md)

## Crypttool

The crypttool is a small commandline utility that allows a user to perform some
simple operations with regards to secrets in a PAAS.

Basic functionality includes sub-commands for `encrypt`, `decrypt` and `reencrypt`
with regards to PAAS related secrets.

The `reencrypt` sub-command specifically will parse the yaml/json file for a PAAS,
decrypt the SSH secrets with the previous private key, reencrypt with the new public
key & write back the PAAS configuration to the file in either yaml or json format.

This will allow for key rotation.

You can also run the `check-paas` sub-command to "check" the PAAS, which means
it will parse the yaml/json file for a PAAS, decrypt the SSH secrets and display
their length and checksums.

Lastly it can be used to `generate` a new public/private keypair.

## Groups [openshift]

Access to a PAAS is granted to specific groups, which can be listed in the PAAS'
configuration file.

A group can contain roles that allow them certain permissions, users and/or an
LDAP query. When configured, the LDAP query is used to find the members of the
group, *in addition* to any users listed specifically in the PAAS configuration.

Please be aware that this is an OpenShift specific feature.

## ManagedByPaas

This is a field in the PAAS configuration, and feature, that allows the user to
indicate that this PAAS is actually managed by another PAAS' ArgoCD.

## Namespace / PaasNS

Namespaces can be used to define extra namespaces to be created as part of this
PAAS project.

## Quotas

There a various quotas that can be configured, but essentially they are: cluster
wide, per PAAS or for a capability.

## Requestor

The requestor is an informational field in the PAAS configuration, which can contain
a string that is intended to point to the person or group responsible for the
application / PAAS.

This could be an ITIL group, Service desk name, email address or any random string.

## SSH Secrets

You can add SSH keys, which are a type of secret, to your PAAS for ArgoCD to use
so it can access a git repository. For example on a self-hosted Github or BitBucket
instance.

The SSH secrets must be encrypted with the public key corresponding to the
private key that was deployed together with the PAAS operator.

## Webservice

The webservice exposes a `/v1/encrypt` endpoint that allows a user to encrypt a
secret using that cluster's private key.

Apart from the encrypt endpoint, some standard endpoints like `/healthz`, `/readyz`,
`/version` and `/metrics` are exposed.
