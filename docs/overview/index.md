---
title: Overview
summary: A short introduction.
authors:
  - hikarukin
date: 2024-07-01
---

# Introduction to the Paas Operator

In a micro-service environment organizations can easily build and maintain thousands of apps.
For each app there is a development process which consists of many types of technologies utilizing many types of resources.

Some examples include:

- Git repositories with code, configuration, documentation, etc.
- CI infrastructure
- CD infrastructure,
- image repositories to hold image artifacts
- the actual namespace running the end application
- and more

Many of these artifacts can be deployed separately for every app, and would then run in their own namespace.

The idea behind the Paas Operator is to bring all of these many pieces of the development process
together in a single context we like to call a 'Project as a Service', e.a. Paas.
The Paas operator then can be used to define a Paas for every App, and will deploy all the required artifacts accordingly.
On top of that, the Paas operator implements multi tenancy between the many Paas resources.

Which means that, by leveraging the Paas operator, an organization can:

- bring together all resources belonging to an App into a single unit called a Paas
- maintain multi tenancy between Paas instances
- enable developers with capabilities to be used as part of the process behind maintaining the App

This documentation site is arranged into a generic section called overview, a user section, an administrator section, and a developer section.
The [Core Concepts](./core_concepts/) pages in the overview section are usually a good starting point.

If you have any questions or feel that certain parts of the documentation can be improved or expanded,
feel free to create a [PR](https://github.com/belastingdienst/opr-paas/pulls) (Pull Request).

For full contribution guidelines, see the `CONTRIBUTING.md` file in the root of
the repository, the [About >> Contributing](/about/contributing/) section and/or the [Development Guide](/development-guide/).
