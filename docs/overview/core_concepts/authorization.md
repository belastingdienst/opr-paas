# Authorization

The whole idea is to create a multi tenancy solution which allows DevOps teams
to request a context for their project, which we like to call a 'Project as a Service',
e.a. Paas.

Requestors of a Paas have the option to setup permissions for groups. Groups can
get permissions on namespaces, and additionally get access in the ArgoCD deployed
for their Paas. Additionally default groups can get permissions on these ArgoCDs as well.

Configuring authorization is done by:

- Cluster administrators defining role mappings in the Paas operator configmap;
- DevOps engineers specifying groups in their Paas resources;
- DevOps engineers can specify groups in their PaasNs resources;
- For every PaasNs the PaasNs controller derives the required RoleBindings and
  creates as required;
  - If a list is specified in the PaasNs it is correlated to the Paas;
    when not defined all groups as specified in the Paas are used by default.
  - For every group, the Paas definition is checked for the functional roles;
    when not defined the default role mapping is used.
  - For every functional role the technical roles are derived from the Paas ConfigMap;
  - For every PaasNs namespace the PaasNs controller creates a role binding for
    every applicable technical role, and adds the groups that should have the
    required permissions;
- For the `argocd` capability, the PaasNs controller adds the required permissions
  to the RBAC block so that the applicable groups get the permissions in ArgoCD
  as required;

  Additionally the Paas operator configmap can have additional `argopermissions` to
  be handed to additional groups (e.a. cluster admins).

## Config examples

### Paas Operator ConfigMap

The Paas Operator ConfigMap (managed by cluster admins) can be configured as follows:

!!! example

    ```yaml
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: PaasConfig
    metadata:
      name: opr-paas-config
    spec:
      argopermissions:
        resource_name: argo-service
        role: admin
        header: |
          g, system:cluster-admins, role:admin
      rolemappings:
        # All groups defined in a Paas without any roles will have the `default`
        # functional role which maps to the OpenShift ClusterRole called view
        default:
          - view
        # All groups defined in a Paas with the `edit` functional role will have a
        # RoleBinding for the ClusterRoles `edit`, `alert-routing-edit`, and
        # `monitoring-edit`
        edit:
          - edit
          - alert-routing-edit
          - monitoring-edit
        # All groups defined in a Paas with the `view` functional role will have a
        # RoleBinding for the ClusterRoles `view`
        readonly:
          - view
        # All groups defined in a Paas with the `admin` functional role will have
        # a RoleBinding for the ClusterRoles `admin`, `alert-routing-edit`, and
        # `monitoring-edit`
        admin:
          - admin
          - alert-routing-edit
          - monitoring-edit
      # Required fields with placeholder values
      capabilities:
        example-capability:
          applicationset: example-appset
          default_permissions: {}
          extra_permissions: {}
          quotas:
            clusterwide: false
            defaults: {}
            min: {}
            max: {}
            ratio: 0
      decryptKeyPaths:
        - /path/to/decrypt/key
      exclude_appset_name: placeholder-appset-name
    ```

!!! Note
Groups that only have view defined will have the same permissions as groups
without any functional roles.

### Paas

Devops engineers could create a Paas with the following definition:

!!! example

    ```yaml
    ---
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: Paas
    metadata:
      name: my-paas
    spec:
      requestor: my-team
      groups:
        # An OpenShift group called `us` is created, and `me` and `you` are added to this group.
        # `us` group has default permissions
        us:
          users:
            - me
            - you
          roles:
            - admin
            - edit
            - view
        # An OpenShift group called `them` is created, and `friend` is added to this group.
        them:
          users:
            - friend
          # `them` group has view permissions
          roles:
            - view
      capabilities:
        # For all capability namespaces (e.a. my-paas-argocd), there will be RoleBindings
        # for `admin`, `edit`, `alert-routing-edit`, and `monitoring-edit`
        argocd:
          enabled: true
      # For all user namespaces (my-paas-cicd, my-paas-test, and my-paas-prod), there
      # will be RoleBindings for `admin`, `edit`, `alert-routing-edit`, and `monitoring-edit`
      namespaces:
        - cicd
        - test
        - prod
      quota:
        limits.cpu: "40"
    ```

With this example (combined with the operator config example), the following would apply:

- In all namespaces (`my-paas-cicd`, `my-paas-test`, `my-paas-prod` and `my-paas-argocd`),
  there will be RoleBindings for `admin`. They will all contain the groups `us` group;
- For all namespaces (`my-paas-cicd`, `my-paas-test`, `my-paas-prod` and `my-paas-argocd`),
  there will be RoleBindings for `view`. They will all contain the groups `us`, and `them`;
- The `argocd-service` deployed in `my-paas-argocd` will have admin permissions set
  for all users in group `us` and group `them`;

!!! Note
That there is also options to manage users with a federated solution.
For more information, see [ldap integration with groupsynclist](groupsynclist.md).

### PaasNs

DevOps engineers could additionally create a PaasNs with the following definition:

!!! example

    ```yaml
    ---
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: PaasNs
    metadata:
      # The name of the resulting namespace would be my-paas-adminonly ([paas name]-[paasns name])
      name: adminonly
      namespace: my-paas-argocd
    spec:
      Paas: my-paas
      # The namespace would only contain RoleBindings for the `us` group, which drills
      # down to the `admin`, `edit`, `view`, `alert-routing-edit`, and `monitoring-edit` ClusterRoles.
      groups:
        - us
    ```

## Caveats

- All groups will have the permissions as specified in the Paas.
- The RBAC block of the Paas argocd capability can be externally managed.
  That also means that changing the groups, operator config,, etc. does not automatically
  apply to existing ArgoCD deployments.
- Next to permissions on groups and users, there is also capabilities to implement
  permissions for service accounts. See [extra_permissions](../../administrators-guide/capabilities.md#configuring-permissions) for
  more info.
- For ldap integration, the operator has options to manage groups using a groupsynclist
  implementation. For more information, see [ldap integration with groupsynclist](groupsynclist.md).
