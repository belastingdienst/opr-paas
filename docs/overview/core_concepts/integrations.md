# OpenShift

The whole idea we tried to solve with the Paas operator, is to create a multi tenancy
solution which allows DevOps teams to request a context for their project, which
we like to call a 'Project as a Service', e.a. Paas.

This aligns heavily with large clusters servicing multiple DevOps teams, which
aligns closely with how we see other organizations running OpenShift.

For other deployments we mostly see small (nearly vanilla) K8S deployments, where
each cluster is only servicing one Devops team specifically. However, we do also
feel that having a single interface to consume features like user management,
capabilities, and quota management could be helpful to have in non-OpenShift
environments too.

## OpenShift specific dependencies

We rely on OpenShift for the following features:

- Cluster Wide Quotas, which seems to be built into the core of OpenShift and does
  not seem to have a k8s generic alternative. Running on vanilla K8S, we would
  probably leave options to have one quota for multiple namespaces and implement
  normal ResourceQuota definitions instead.
- We currently rely on the Groups implementation in OpenShift. We are revisiting
  the current architecture and will work towards a solution that can work natively
  in K8S as good as possible.

## ArgoCD

The Paas Operator can optionally integrate with ArgoCD’s ApplicationSets through a plugin-generator service.
The goal here is to deploy the content of Paas capabilities through ApplicationSets. The plugin-generator can
be used to dynamically provide parameters for the capability related ApplicationSets.

This integration is **not required** for using the Paas Operator but can simplify workflows where ArgoCD
is used to manage deployments for Paas capabilities.

More information about the Plugin-Generator can be found upstream in
the [ArgoCD documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/Generators-Plugin/)

### What users can expect

- **Dynamic parameter generation** – No need to hardcode parameter lists in your ApplicationSets; the plug-in fetches them directly
  from the Paas Operator at runtime.
- **Always up-to-date** – Parameters reflect the **current state of Paas resources** in the cluster.
- **Aligned with Paas capabilities** – Output is based on your configured capabilities and the custom fields defined in
  `PaasConfig`.
- **Optional integration** – If you do not enable the plug-in server, the Paas Operator will still function normally; you just won’t
  have this automated ApplicationSet parameter population.

### What it exposes

The plug-in provides an HTTP endpoint that ArgoCD calls with:

- The **ApplicationSet name** (`applicationSetName`)
- A set of **input parameters**

**Expected input parameter:**  
`capability` – The name of a capability, which is defined in your `PaasConfig`.

When provided, the plug-in will:

- List all Paas resources in the cluster that have the given capability enabled.
- Extract the **custom field values** for each resource as defined in `PaasConfig`.
- Return these as individual maps in the `parameters` array of the response.

Example request:

```json
{
	"applicationSetName": "my-capability-appset",
	"input": {
		"parameters": {
			"capability": "my-capability"
		}
	}
}
```

Example response:

```json
{
	"output": {
		"parameters": [
			{
				"team": "dev-team",
				"env": "staging"
			},
			{
				"team": "qa-team",
				"env": "production"
			}
		]
	}
}
```

Here, `team` and `env` are _custom fields_ defined in your `PaasConfig`, and their values come from the corresponding Paas objects
in the cluster.

If no matches are found, parameters will be returned as an empty array `([])`.

### How to enable the plug-in

The plug-in server is only started if both of the following are configured:

1. Bind address flag – Determines where the plug-in HTTP server listens:

```bash
--argocd-plugin-generator-bind-address=:4355
```

If omitted, the plug-in generator is not added to the operator.

2. Authentication environment variable – The server requires a bearer token for incoming requests:

```bash
export ARGOCD_GENERATOR_TOKEN=your-secret-token
```

ArgoCD must be configured to use this token when calling the endpoint.

### Example ApplicationSet

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: my-capability-appset
spec:
  generators:
    - plugin:
        configMapRef:
          name: argocd-plugin-generator-config
        input:
          parameters:
            capability: my-capability
  template:
    metadata:
      name: '{{team}}-{{env}}'
    spec:
      project: default
      source:
        repoURL: https://github.com/my-org/my-repo.git
        targetRevision: HEAD
        path: apps/{{env}}
      destination:
        server: https://kubernetes.default.svc
        namespace: default
```