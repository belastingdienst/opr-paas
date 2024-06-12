---
title: End to end testing
summary: How we do end to end testing.
authors:
  - hikarukin
date: 2024-08-21
---

# Introduction

## Running the tests

Run the tests using the following command: `go test -v ./test/e2e`

## Design considerations

We've decided to use the e2e-framework from K8S. The advantages of this framework
are that any connection to a K8S cluster can be used to execute these tests against
that cluster.

This makes the tests loosely coupled, and thus usable against various types of
clusters. For example a K3S cluster spun up on a developer's machine or in one in
GitHub actions.

We can use our favorite programming language to write the tests. The framework
uses a kubernetes client to execute k8s commands to the connected cluster.
The cluster should have a PaaS operator installed to reconcile PaaS'es during
the execution of these tests. The tests assert whether the expected resources
are created on the cluster.

## Setup

The host running these tests, must have an active connection to a k8s cluster in
it's kubeConfig. It must be logged in and have the appropriate permissions to
apply the resources used in this test.

The tests, by default, run in a namespace: paas-e2e which will be created during
test setup (main_test.go) and deleted afterward. If you would like to use an
existing namespace, set the environment variable: `PAAS_E2E_NS` to the namespace
name.

Note: the tests do not create the namespace for you in case it happens to be
missing, so make sure to create it or be prepared to enjoy the error message.