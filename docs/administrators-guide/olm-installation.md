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
- `fast`: intended for OTA or pre-production clusters;
- `stable`: intended for production clusters.

New releases are first added to `candidate`. Promotion to `fast` and `stable`
is performed later, after validation in the earlier environment.

# How rollout works

The operator publishes:

- one immutable operator image per release;
- one immutable OLM bundle image per release; and
- one catalog image built from the source-controlled file-based catalog in this repository.

Rollout is controlled by OLM channel membership, not by separate catalog tags per
environment. All clusters can use the same catalog image reference while subscribing
to different channels.

This means:

- development clusters subscribe to `candidate`;
- OTA clusters subscribe to `fast`;
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
spec:
  targetNamespaces:
    - paas-system
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

For OTA clusters, change the channel to `fast`.
For production clusters, change the channel to `stable`.

# Airgapped environments

In airgapped environments, mirror the following images into the registry that
your cluster can access:

- the operator image;
- the OLM bundle image; and
- the OLM catalog image.

Mirroring is the transport mechanism. Rollout policy still comes from the OLM
channel selected in the `Subscription`.
