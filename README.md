# opr-paas
Deze operator is bedoeld om PaaS resources te kunnen reconcilen naar namespaces, cluster quota,s ldap groepen, etc.

## IMPORTANT
The config folder is in this repo as an example only.
The actual argocd config is maintained in the opr-paas-config repo.
When changing the crd, first run `make manifests` in the root of this repo.
Then copy config/crd/bases/cpet.belastingdienst.nl_paas.yaml to the opr-paas-config repo and dsitribute with ArgoCD from there

## Description
Het idee is dat onze klanten een PaaS resource kunnen aanmaken en dat ze hiermee een applicatieomgeving kunnen (laten) creeren.
de applicatie omgeving omvat:
- argocd (namespace, quota, argocd CR
- ci (namespace, quota, tekton dingen)
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
For now this is a Belastingdienst Internal project.
We might Open Source it in the future, or we might not.

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

