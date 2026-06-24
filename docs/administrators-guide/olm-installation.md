---
title: OLM Installation
summary: Install and operate the Paas Operator through OLM using channel-based rollout.
authors:
  - CtrlShiftOps
date: 2026-04-15
---

# Introduction

The Paas Operator can be installed through Operator Lifecycle Manager (OLM).
This installation model is intended for environments where you want:

- a managed operator lifecycle through OLM;
- staged rollout across multiple environments; and
- a source-controlled catalog with explicit promotion between channels.

The project uses the following OLM channels:

- `candidate`: intended for development clusters;
- `fast`: intended for pre-production clusters;
- `stable`: intended for production clusters.

New releases are first added to `candidate`. Promotion to `fast` and `stable`
is performed later, after validation in the earlier environment.
Promotion is sequential: a version must first be promoted to `fast` before it
can be promoted to `stable`.

# How rollout works

The operator publishes:

- one immutable operator image per release;
- one immutable OLM bundle image per release; and
- one catalog image built from the source-controlled file-based catalog in this repository.

Rollout is controlled by OLM channel membership. All clusters can use the same catalog image reference while subscribing
to different channels.

This means:

- development clusters subscribe to `candidate`;
- pre-production clusters subscribe to `fast`;
- production clusters subscribe to `stable`.

# Catalog image

The OLM catalog image is published to:

```text
ghcr.io/belastingdienst/opr-paas-catalog:latest
```

This tag is only a delivery reference for the current catalog image. The actual
rollout behavior comes from the subscribed OLM channel.

# Example installation objects

You will typically need:

- a `CatalogSource`;
- an `OperatorGroup`; and
- a `Subscription`.

## CatalogSource

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: opr-paas-catalog
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: ghcr.io/belastingdienst/opr-paas-catalog:latest
  displayName: opr-paas Catalog
  publisher: Belastingdienst
```

## OperatorGroup

```yaml
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: opr-paas
  namespace: paas-system
spec: {}
```

## Subscription

Development clusters should subscribe to `candidate`:

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: opr-paas
  namespace: paas-system
spec:
  channel: candidate
  installPlanApproval: Automatic
  name: opr-paas
  source: opr-paas-catalog
  sourceNamespace: openshift-marketplace
```

For pre-production clusters, change the channel to `fast`.
For production clusters, change the channel to `stable`.

To enable the ArgoCD plugin generator when installing through OLM, set
`ARGOCD_PLUGIN_GENERATOR_BIND_ADDRESS` through `spec.config.env` on the
`Subscription`, and provide `ARGOCD_GENERATOR_TOKEN` through the operator pod
environment.

Metrics remain disabled by default. For a secure HTTPS endpoint, set
`METRICS_BIND_ADDRESS=:8443` and `METRICS_SECURE=true` through
`spec.config.env` on the `Subscription`. The scraper must be configured for
TLS and Kubernetes authentication and authorization. When no metrics
certificate is configured, controller-runtime generates a self-signed
certificate.

For a trusted in-cluster scraper, metrics can instead be exposed over HTTP by
setting `METRICS_BIND_ADDRESS=:8080` and `METRICS_SECURE=false`. Do not expose
this endpoint outside the trusted cluster network. Command-line flags continue
to take precedence over these environment-backed defaults.

The OLM-managed operator runs two replicas by default to keep the admission
webhooks available during pod disruptions and rolling updates. Leader election
ensures that only one replica actively runs the controllers at a time.

# Airgapped environments

In airgapped environments, mirror the following images into the registry that
your cluster can access:

- the operator image;
- the OLM bundle image; and
- the OLM catalog image.

Mirroring is the transport mechanism. Rollout policy still comes from the OLM
channel selected in the `Subscription`.
