---
title: Maintenance
summary: Introduction to contributing to the Paas operator.
authors:
  - devotional-phoenix-97
  - hikarukin
  - CtrlShiftOps
date: 2024-07-04
---

Introduction
============

This file documents the methods and standards that should be applied by the maintainers
of this project. For example: how to create a new release.

Standards used
--------------

### Commits

We adhere to the [Conventional Commits v1.0](https://www.conventionalcommits.org/en/v1.0.0/)
standard.

### Versioning

For versioning purposes, we adhere to the [SemVer v2.0.0](https://semver.org/spec/v2.0.0.html)
standard with a side note that we always prefix the semantic version number with
the character 'v'. This stands for "version".

As a quick summary, this means we use version numbers in the style of vX.Y.Z.
With:

- X being major, including breaking, changes;
- Y being minor, possibly including patch but never breaking, changes;
- Z being a patch, but never breaking, changes;

Methods used
------------

### Creating a Release

We release from `main`. All changes to `main` are made through PRs. Merging a PR
triggers the release drafter action to create a draft release.

The process to create a release is mostly automated. To start it:

* Merge one or more PRs to `main`;
* Ensure completeness;
* Edit the draft release and publish it.

Publishing the release triggers the automated release workflows. In particular:

- the operator image is built and published as `ghcr.io/belastingdienst/opr-paas:vX.Y.Z`;
- the `install.yaml` artifact is generated and attached to the GitHub release;
- one immutable OLM bundle image is built and published as `ghcr.io/belastingdienst/opr-paas-bundle:vX.Y.Z`;
- the source-controlled file-based catalog under `catalog/` is updated to add
  the release to the `candidate` channel;
- that catalog change is committed back to the default branch;
- a catalog image is built from the committed catalog source and published.

The catalog image is published using:

- an immutable `git-<sha>` tag for traceability; and
- the moving `latest` tag as the current catalog delivery reference.

The rollout semantics for OLM do **not** come from image tags. They come from
OLM channel subscriptions:

- `candidate` for development clusters;
- `fast` for OTA or pre-production clusters; and
- `stable` for production clusters.

#### OLM promotion flow

OLM promotion is handled through the `Promote OLM catalog` workflow.

This workflow:

- reuses the already published immutable bundle image for the requested release;
- updates the source-controlled catalog metadata in `catalog/`;
- promotes that version into `fast` or `stable`;
- commits the catalog change back to the default branch; and
- rebuilds and republishes the catalog image from that committed source.

Promotion does not rebuild the bundle. It only changes catalog metadata.

#### Operational model

The intended release flow is:

1. Publish a GitHub release.
2. The release workflow publishes the immutable operator and bundle images.
3. The same workflow updates the source-controlled catalog and adds the new release
   to `candidate`.
4. Development clusters subscribed to `candidate` can upgrade.
5. After validation, run the promotion workflow to add the same bundle version to `fast`.
6. After validation in OTA, run the promotion workflow again to add the same bundle
   version to `stable`.

No new bundle is built during promotion. Promotion changes catalog metadata only.

#### Important: No Backports Policy

We do not support backports to previous release versions.
Fixes are only provided for the **current main release series**. For example, if
version 2.x.x is the active release series, no fixes will be made or backported
to the 1.x.x line.

We follow a **roll-forward support model**:

* All users are expected to upgrade to the latest available release to receive fixes.
* Fix releases (patch versions) are provided for the current main release series only.
* Users remaining on older versions do so at their own risk, as we do not provide
  maintenance or security updates for them.

This approach allows us to focus our efforts on improving the latest version without
fragmenting support across multiple release lines.

---

### Creating a Hotfix Release

Hotfix releases are created from the relevant tag. The process is similar to
creating a regular release.

The process is as follows:

* Create a new branch based on the **latest release tag of the current main release series**
  that needs the fix;
* Merge one or more PRs to this branch;
* Ensure completeness;
* Edit the draft release and publish it;
  Ensure the release only contains the hotfix!
* Merge the hotfix branch back into `main` to keep `main` up to date.

> **Note:** Hotfix releases are only supported for the actively maintained release
  series. We do not create hotfixes for previous major versions.
