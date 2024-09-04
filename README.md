# opr-paas

## Goal

The PaaS operator delivers an opinionated 'Project as a Service' implementation where
development teams can request a 'Project as a Service' by defining a PaaS resource.

A PaaS resource is used by the operator as an input to create namespaces limited
by Cluster Resource Quota's, granting groups permissions and (together with a clusterwide
ArgoCD) creating capabilities such as:

- a PaaS specific deployment of ArgoCD (continuous deployment);
- Tekton (continuous integration);
- Grafana (observability); and
- KeyCloak (Application level Single Sign On);

A PaaS is all a team needs to hit the ground running.

## Quickstart

Deploy the operator using the following command:
```
kubectl apply -f \
  https://raw.githubusercontent.com/belastingdienst/paas/release-1.0.0/releases/opr-paas-1.0.0.yaml
```

This will create:

- a namespace called `paas-system`;
- 2 CRDs (`PaaS` and `PaasNs`);
- a service account, role, role binding, cluster role and cluster role binding for
  all permissions required by the operator;
- a viewer & an editor cluster role for PaaS and PaasNs resources;
- a configmap with all operator configuration options;
- a secret with a newly generated key pair used for;
- a deployment running the operator and a deployment running an encryption service;

Feel free to change config as required.

### Change configuration
The quick start yaml file is a result from parts of the config folder, which is
in this repo as an example only. It is advised to copy it to a config repo and use
that to maintain your own deployment.

When changing the crd, first run `make manifests` in the root of this repository.

Then copy `config/crd/bases/cpet.belastingdienst.nl_paas.yaml` to the `opr-paas-config` repo and distribute with ArgoCD from there.

## Description
We want our developer teams to be able to create application environments with
great ease and provide PaaS as an interface for this. Application environments consist of:

- argocd (namespace, quota, argocd CR);
- ci (namespace, quota, tekton example pipelines and tasks, etc.);
- SSO (namespace, quota, keycloak);
- Grafana (namespace, quota, Grafana);

## Background information
- [build-kubernetes-operator-six-steps](https://developers.redhat.com/articles/2021/09/07/build-kubernetes-operator-six-steps#setup_and_prerequisites)
- [operator sdk installation instructions](https://sdk.operatorframework.io/docs/installation/)

## Contributing

Please refer to our documentation in the [CONTRIBUTING.md](./CONTRIBUTING.md) file and the developer section of the documentation site if you want to help us improve the PaaS solution.

## License

Copyright 2024, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.

See [LICENSE.md](./LICENSE.md) for details.
