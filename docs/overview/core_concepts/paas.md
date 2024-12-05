# Paas

The whole idea is to create a multi tenancy solution which allows DevOps teams
to request a context for their project, which we like to call a 'Project as a Service',
e.a. Paas.

The Paas is a higher level construct, which consists of many parts, including:

- namespaces;
- Quotas;
- authorization;
- capabilities;

DevOps teams request this Paas context by defining a Paas resource through the K8S API.

At the very least a Paas resource has the following defined:

- `apiVersion`, kind (as needs to be defined for every other k8s resource)
- `metadata.name`, which is unique (cluster-wide)
- `spec.requestor`, which is an informational field representing the requestor of
  this Paas, for administrative purposes
- `quota`, which sets the amount of quota for all namespaces that are part of
  this Paas (capability namespaces excluded)

Additionally the following optional settings can also be defined:

- `capabilities`, which can be used to enable Paas extensions such as an ArgoCD to
  manage all Paas namespaces, Grafana to monitor Paas namespaces, etc. More information
  can be found in our [capabilities](capabilities.yaml) documentation.
- `spec.sshSecrets`, which can be used to seed secrets that ArgoCD requires for
  accessing repositories. See [sshSecrets](sshsecrets.yaml) for more information.
- `spec.groups`, which can be used to configure authorization. See [authorization](authorization.yaml)
  for more information.
- `spec.namespaces`, which can be used to define namespaces as part of the Paas.
  Alternatively, they can be manually defined as [PaasNs](PaasNs.yaml) resources.

## Example Paas

!!! example

    ```yaml
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: Paas
    metadata:
      name: my-paas
    spec:
      capabilities:
        # Enable argocd
        argocd:
          enabled: true
          # Bootstrap application to point to the root folder
          gitPath: .
          # Bootstrap application to point to the main branch
          gitRevision: main
          # Bootstrap application to point to this repo
          gitUrl: "ssh://git@github.com/belastingdienst/my-paas-repo.git"
        # enable grafana
        grafana:
          enabled: true
          quota:
            limits.cpu: "5"
            limits.memory: "2Gi"
    ```

!!! Notes
    Labels defined on Paas resources are copied to child resources such as PaasNs,
    quotas, groups, ArgoApps, ArgoProjects, etc.
      
    The only exception is the `app.kubernetes.io/instance`.
