---
title: Setting up your development environment
summary: A short manual on how to setup your development environment
authors:
  - Devotional Phoenix
date: 2023-12-21
---

# Introduction

Currently the PaaS operator uses Code Ready Containers for testing. The
operator-sdk uses KIND instead of CRC. [KIND](https://sigs.k8s.io/kind) is lower
in resource consumption, but also lacking a lot we use, which is shipped by
default in OpenShift.

Future releases might be more kind to [KIND](https://sigs.k8s.io/kind), in which
case will update this chapter.

## Installation of CRC
You’ll need a Kubernetes cluster to run against. We currently use Code Ready
Containers.

Installation instructions can be found at [Red Hat](https://console.redhat.com/openshift/create/local).

## Starting CRC
If you need to start CRC (like after a reboot, which is rerquired on BD DBO for some weird unknown reason), run `crc start` in a terminal other than iterm2 (like terminal or kitty).
After that login using the oc command as display'ed in output of `crc start`.

### Running the operator

1. Install Custom Resource examples:

```sh
oc apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:

```sh
MYGHPROJ=user1/paas
make docker-build docker-push IMG=ghcr.io/${MYGHPROJ}/opr-paas:latest
```

3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
MYGHPROJ=user1/paas
make deploy IMG=ghcr.io/${MYGHPROJ}/opr-paas:latest
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller from the cluster:

```sh
make undeploy
```
