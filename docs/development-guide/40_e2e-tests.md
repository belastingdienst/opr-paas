---
title: End to end testing
summary: How we do end to end testing.
authors:
  - hikarukin
date: 2024-08-21
---

# Introduction

We have a bunch of end-to-end tests that can be used by you to verify whether the
operator still works as expected after you made any code changes. These tests are
also part of our continues integration pipeline on GitHub.

## Prerequisites

Ensure you have a vanilla Kubernetes or OpenShift cluster running. We can heartily
recommend using [kind](https://kind.sigs.k8s.io).

!!! example

    ```kind create cluster``` 

## Running the tests

1. In case of a vanilla kubernetes cluster, run: `make setup-e2e` <br/>
  This will apply mocks, etc. needed to run the operator.
2. Start the operator: `make run`
3. Finally, run the actual e2e tests: `make test-e2e`

## Design considerations

We've decided to use the e2e-framework from K8S. The advantages of this framework
are that any connection to a K8S cluster can be used to execute these tests against
that cluster.

This makes the tests loosely coupled, and thus usable against various types of
clusters. For example a K3S cluster spun up on a developer's machine or in one in
GitHub actions.

We can use our favorite programming language to write the tests. The framework
uses a kubernetes client to execute K8S commands to the connected cluster.
The cluster should have a Paas operator installed to reconcile Paas'es during
the execution of these tests. The tests assert whether the expected resources
are created on the cluster.

## Setup

The host running these tests, must have an active connection to a K8S cluster in
it's kubeConfig. It must be logged in and have the appropriate permissions to
apply the resources used in this test.

The tests, by default, run in a namespace: `paas-e2e` which will be created during
test setup (`main_test.go`) and deleted afterward. If you would like to use an
existing namespace, set the environment variable: `PAAS_E2E_NS` to the namespace
name.

!!! Info
    The tests do not create the custom namespace for you in case it happens to
    be missing, so make sure to create it or be prepared to enjoy the error message.