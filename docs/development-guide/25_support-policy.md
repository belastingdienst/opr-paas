---
title: Support Policy
summary: The support policy adhered to by this project.
authors:
  - hikarukin
date: 2025-06-23
---

# Support Policy

By enforcing this support and versioning strategy, we can focus on delivering timely,
reliable updates and maintaining a streamlined release process for all users.

We follow a **roll-forward-only support model**. This ensures that all users
benefit from the latest improvements, fixes, and security updates without fragmenting
support across outdated versions.

## Supported Versions

- We only support the **latest active major release series.**
- Fixes, improvements, and security updates will only be applied to the most recent
  release.
- Previous major release series are considered **end-of-life (EOL)** and will not
  receive backported fixes or patches.
- Users are expected to upgrade to the latest available version to remain supported.

## No Backports

- **We do not backport fixes to older versions.**
- If an issue is identified in an older release, the resolution will be provided
  in a new release based on the current active version.
- Hotfixes and patches are provided only within the current supported release series.

## Hotfixes

- Hotfixes are supported **exclusively** for the currently supported release series.
- Hotfixes are created from the latest release tag.
- All hotfix branches must be merged back into `main` after release to ensure continuity.

## Versioning

We adhere to [Semantic Versioning v2.0.0](https://semver.org/spec/v2.0.0.html) with
the following conventions:
- All version numbers are prefixed with the letter **'v'** (e.g., `v2.1.0`).
- **vX.Y.Z** structure:
  - **X**: Major version – may introduce breaking changes.
  - **Y**: Minor version – adds functionality in a backward-compatible manner.
  - **Z**: Patch version – backward-compatible fixes.

> **Note:** Only the latest major version is supported. Minor and patch releases
  within the current major version are eligible for updates.

## Commits

We strictly follow the [Conventional Commits v1.0.0](https://www.conventionalcommits.org/en/v1.0.0/)
specification. This ensures that commit messages are:
- Clear and machine-readable.
- Aligned with semantic versioning for automated changelog generation and release drafting.
