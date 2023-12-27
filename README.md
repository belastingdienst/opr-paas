# opr-paas

## Goal

The PaaS operator delivers an opiniated 'Project as a Service' implementation where
development teams can request a 'Project as a Service' by defining a PaaS resource.

A PaaS resource is used by the operator uses as an input to create namespaces
limited by Cluster Resource Quota's, granting groups permissions and (together
with a clusterwide ArgoCD) creating capabilities such as a PaaS specific deployment
of ArgoCD (continuous deployment), Tekton (continuous integration), Grafana (observability),
and KeyCloak (Application level Signle Sign On).

A PaaS is all a team needs to hit the ground running.

## Quickstart

Deploy the operator using the following command:
```
kubectl apply -f \
  https://raw.githubusercontent.com/belastingdienst/paas/release-1.0.0/releases/opr-paas-1.0.0.yaml
```

This will create:
- a namespace called paas-system
- 2 CRD's (PaaS and PaasNs)
- a service account, role, rolebinding, clusterrole and clusterrolebinding for all permissions required by the operator
- a viewer and an editor clusterrole for PaaS and PaasNs resources
- a configmap with all operator configuration options
- a secret with a newly generated keypair used for
- a deployment running the operator and a deployment running an encryption service

Feel free to change config as required.

### Change configuration
The quickstart yaml file is a result from parts of the config folder, which is in this repo as an example only.
It is adviced to copy it to a config repo and use that to maintain your own deployment.

When changing the crd, first run `make manifests` in the root of this repo.
Then copy config/crd/bases/cpet.belastingdienst.nl_paas.yaml to the opr-paas-config repo and dsitribute with ArgoCD from there

## Description
We want our developer teams to be able to create application environments with great ease and provide PaaS as an interface for this.
Application environments consist of:
- argocd (namespace, quota, argocd CR
- ci (namespace, quota, tekton example pipelines and tasks, etc.)
- SSO (namespace, quota, keycloak)
- Grafana (namepace, quota, Grafana)

## Background information
- [build-kubernetes-operator-six-steps](https://developers.redhat.com/articles/2021/09/07/build-kubernetes-operator-six-steps#setup_and_prerequisites)
- [operator sdk installation instructions](https://sdk.operatorframework.io/docs/installation/)

## Getting Started

### Instalation CRC
You’ll need a Kubernetes cluster to run against.
We run on Code Ready Containers. Instructions:
- [Red Hat](https://console.redhat.com/openshift/create/local)

**Note** operator-sdk uses KIND instead of CRC. [KIND](https://sigs.k8s.io/kind) is lower in resource consumption, but also lacking a lot we use in BD, which is shipped by default in OpenShift.

## Starting CRC
If you need to start CRC (like after a reboot, which is rerquired on BD DBO for some weird unknown reason), run `crc start` in a terminal other than iterm2 (like terminal or kitty).
After that login using the oc command as display'ed in output of `crc start`.

### Running on the cluster
1. Install Instances of Custom Resources:

```sh
oc apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=belastingdienst/opr-paas
```

3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=belastingdienst/opr-paas
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

## Contributing

 Please refer to our documentation on [how to contribute](./CONTRIBUTING.md) if you want to help us improve the PaaS solution.

## License

Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.

See [LICENSE.md](./LICENSE.md) for details.
