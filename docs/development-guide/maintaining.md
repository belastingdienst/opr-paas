---
title: Maintenance
summary: Introduction to contributing to the Paas operator.
authors:
  - devotional-phoenix-97
  - hikarukin
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
- Z being patch, but never breaking, changes;

Methods used
------------

### Creating a Release

We release from `main`. All changes to `main` are made through PRs. Merging a PR
triggers the release drafter action to create a draft release.

The process to create a release is mostly automated. To start it:

* Merge one or more PRs to `main`;
* Ensure completeness;
* Edit the draft release and publish it.

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
