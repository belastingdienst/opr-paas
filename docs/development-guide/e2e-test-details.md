# E2E test cases for opr-paas

Below you will find a list of features that we want to test using end-to-end testing.
For each feature, the setup and the assessments are listed.

TODO:
1. Take up `RoleMappings` in `testConfig`;
2. Test `extra_permissions` / `default_permissions` in config;

## Paas

What we test: CRUD for Paas

Scenarios:

1. A `Paas` is created that already exists.<br/><br/>
   **Given** that a specified `Paas` already exists,<br/>
   **when** someone configures a new `Paas` with the same name,<br/>
   **then** the `Paas` related namespace must not be created<br/>
   **and** the operator must return an error.

2. A minimal `Paas` is created.<br/><br/>
   **Given** that the `Paas` does not exist,<br/>
   **when** someone configures the minimal `Paas`<br/>
   **then** the `Paas` namespace must be created<br/>
   **and** the status of the `Paas` contains no errors.

3. A `Paas` is renamed.<br/><br/>
   **Given** that a specified `Paas` exists,<br/>
   **when** the `Paas` is renamed in the configuration,<br/>
   **then** the related `Paas` namespace must be renamed as well.

4. A `Paas` is deleted.<br/><br/>
   **Given** that a specified `Paas` exists,<br/>
   **when** the `Paas` is deleted,<br/>
   **then** the namespace belonging to the `Paas` is also removed.

Post scenarios: reset environment to clean slate.

## PaasNs

What we test: CRUD for PaasNs

Scenarios:

1. A `PaasNs` is created for a `Paas` that does not exist.<br/><br/>
   **Given** that a specified `Paas` does not exist,<br/>
   **when** someone configures a `PaasNs` to be created under specified `Paas`,<br/>
   **then** the `PaasNs` related namespace must not be created<br/>
   **and** the status of said `PaasNs` must contain the correct error.

2. A minimal `Paas` that is referenced in the `PaasNs` is created.<br/><br/>
   **Given** that the `PaasNs` does not exist,<br/>
   **and** that the minimal `Paas` does not exist,<br/>
   **when** someone configures the minimal `Paas`<br/>
   **and** someone configures the `PaasNs` referencing the minimal `Paas`<br/>
   **then** the `PaasNS` namespace must be created<br/>
   **and** the `PaasNS` namespace has the `quota_label` label with the value of ... FIXME<br/>
   **and** the status of the `PaasNs` contains no errors.

3. A `PaasNs` is renamed.<br/><br/>
   **Given** that a specified `PaasNs` exists,<br/>
   **when** the `PaasNs` is renamed in the configuration,<br/>
   **then** the related `PaasNs` namespace must be renamed as well.

4. A `PaasNs` is deleted.<br/><br/>
   **Given** that a specified `PaasNs` exists,<br/>
   **when** the `PaasNs` is deleted,<br/>
   **then** the namespace belonging to the `PaasNs` is also removed.

Post scenarios: reset environment to clean slate.

## Namespaces

1. A minimal `Paas` configuration without namespaces results in one namespace.<br/><br/>
   **Give** a minimal `Paas` configuration without namespaces,<br/>
   **when** someone adds said configuration to the system,<br/>
   **then** a single namespace should have been created,<br/>
   **and** this namespace should be named the same as the `Paas`,<br/>
   **and** no `PaasNs`'s or there namespaces are linked to th `Paas`.

2. Adding two namespaces to the `Paas`'s spec.<br/><br/>
   Given a minimal `Paas` without any namespaces exists,<br/>
   when the `Paas` is updated by adding 2 namespaces to the spec,<br/>
   then one namespace with the same name as the `Paas` must exist,<br/>
   and two `PaasNs`'s must exist in the namespace of the `Paas`,<br/>
   and these `PaasNs`'s each have a namespace,<br/>
   and these `PaasNs` namespaces must be named according to their `spec.namespaces` entries, prefixed by the `Paas` namespace name

3. Removing the namespaces from a `Paas`.<br/><br/>
   Given a minimal `Paas` with two namespaces exists,<br/>
   when the `Paas` configuration is updated to remove the namespaces,<br/>
   then the `PaasNs`'s should have be removed from the `Paas` namespace,<br/>
   and the `Paas` namespace was removed.

Post scenarios: reset environment to clean slate.

## ClusterResourceQuotas

What we test: CRQ CRUD

!!! Note
    The `spec.quota` does not fall under not cluster wide quotas, hence a separate
    set of test scenarios.

1. Ensure the correct CRQ is created for a `Paas`.<br/><br/>
   **Given** a minimal `Paas` exists,<br/>
   **when** someone adds a quota to `spec.quota` for the `Paas` configuration,<br/>
   **then** a CRQ with the name of the `Paas` must be created,<br/>
   **and** `clusterquotagroup=` followed by the `Paas` name should have been applied
           as label selector on the CRQ,<br/>
   **and** the size of the created CRQ equals the size as specified in the `spec.quota`.

2. The `spec.quota` for a `Paas` is updated.<br/><br/>
   **Given** a minimal `Paas` exists,<br/>
   **and** a valid CRQ exists for this `Paas`,<br/>
   **when** someone updates the `spec.quota` section for the specified `Paas` configuration,<br/>
   **then** the CRQ should be updated,<br/>
   **and** the size of the updated CRQ equals the size as specified in the `spec.quota`.

3. Removing the `Paas` should remove the associated CRQ.<br/><br/>
   **Given** a minimal `Paas` and its associated CRQ exist,<br/>
   **when** the `Paas` is removed,<br/>
   **then** the associated CRW with the name of the `Paas` should have removed as well.

Post scenarios: reset environment to clean slate.

## Cluster wide quotas

What we test: cluster wide quota CRUD

Scenarios:

TODO

Post scenarios: reset environment to clean slate.

## Groups => Users

What we test: managing users and group memberships through `Paas` configuration.

Scenarios:

1. Creating a group with a single user without a specified role.<br/><br/>
   **Given** a minimal `Paas` with a single namespace,<br/>
   **and** a group with a single user without a specified role,<br/>
   **when** that `Paas` is created,<br/>
   **then** a `Group` was created with the correct name,<br/>
   **and** the user is a member of said group,<br/>
   **and** the correct labels were placed on the group,<br/>
   **and** the Owner Reference for the Group points to the correct Paas,<br/>
   **and** there are were no changes to the Whitelist,<br/>
   **and** the rolebinding on the namespace points to the group to the default role,<br/>
   **and** the rolebinding was not applied to the default `Paas` namespace (name of the `Paas`)

2. Updating the `Paas`, adding a group with a role other than default.<br/><br/>
   **Given** an existing, minimal `Paas` with a single namespace,<br/>
   **when** a group is added to said `Paas`,<br/>
   **and** said group has a specific role, other than default (see test_config),<br/>
   **and** a different user is a member of said group than in scenario 1,<br/>
   **then** a `Group` was created with the correct name,<br/>
   **and** the user is a member of said group,<br/>
   **and** the correct labels were placed on the group,<br/>
   **and** the Owner Reference for the Group points to the correct Paas,<br/>
   **and** there are were no changes to the Whitelist,<br/>
   **and** the rolebinding on the namespace points to the group to the default role,<br/>
   **and** the rolebinding was not applied to the default `Paas` namespace (name of the `Paas`)

3. Removing the `Paas`.<br/><br/>
   **Given** an existing `Paas` with a single group,<br/>
   **when** said `Paas` is removed,<br/>

!!! note
   *There are known issues on groups, updating / removing does not go perfectly. Goal here is to test the baseline in the current situation.*

   *Determine what that baseline currently is.*

Post scenarios: reset environment to clean slate.

## Groups => Query

What we test: managing users and group memberships through an LDAP query specified
              in the `Paas` configuration.

Scenarios:

1. Minimal `Paas` with one namespace and a `Group` with a `Query` but no `Role`.<br/><br/>
   **Given** no existing `Paas`,<br/>
   **when** a minimal `Paas` is created with a single namespace,<br/>
   **and** a `Group` with a `Query`, but without a `Role`,<br/>
   **then** a `Group` with the correct name should have been created,<br/>
   **and** there should be no users in said `Group`,<br/>
   **and** the correct labels were added onto the `Group`,<br/>
   **and** the Owner Reference for the Group points to the correct Paas,<br/>
   **and** the query was added to the whitelist,<br/>
   **and** the rolebinding on the namespace points to the group, to the specified role,<br/>
   **and** the rolebinding was not applied to the default `Paas` namespace (name of the `Paas`).
   
2. Updating the `Paas`, adding a group with a role other than default.<br/><br/>
   **Given** an existing, minimal `Paas` with a single namespace,<br/>
   **when** another query is added to said `Paas` (compared to step scenario 1),<br/>
   **and** said group has a specific role, other than default (see test_config),<br/>
   **then** a `Group` was created with the correct name,<br/>
   **and** there are no users in said group,<br/>
   **and** the correct labels were placed on the group, (no ldap things)<br/>
   **and** the Owner Reference for the Group points to the correct Paas,<br/>
   **and** there were changes made to the Whitelist,<br/>
   **and** the rolebinding on the namespace points to the group to the specified role,<br/>
   **and** the rolebinding was not applied to the default `Paas` namespace (name of the `Paas`)

3. Removing the `Paas`.<br/><br/>
   **Given** an existing `Paas` with a single group,<br/>
   **when** said `Paas` is removed,<br/>

!!! note
   *There are known issues on groups, updating / removing does not go perfectly. Goal here is to test the baseline in the current situation.*

   *Determine what that baseline currently is.*

Post scenarios: reset environment to clean slate.

!!! note
    TODO, there a known issue regarding groups. Good regression test to model the following steps:
      1. Create `Paas` with query;
      2. Update `Paas` from step 1, remove query and add users to the group;
      3. Group is not removed from whitelist;
      4. Sync fails because the required `ldap.uid` doesn't match the groupname. (We don't fully test sync.)

## Secrets

*The capabilities are also used to test clusterwide quotas*

### Capability ArgoCD

What we test: creating a `Paas` with the `ArgoCD` capability enabled.

Scenarios:

1. Create a minimal `Paas` with ArgoCD capability enabled;
    1. Assess the list entry exists in the applicationset;
    2. Assess that namespace: `paasnaam-argocd` was created;
    3. Assess that the Argo Application was created in namespaces;
        1. Assess gitUrl, path etc. exist in spec;
        2. Assess RBAC .. determine how;
        3. Assess Secrets exist in namespace and in argo...?
        4. Assess Exclude appset is included in spec as ignoreDiff;
    4. Assess quota
        1. Assess a quota with the name `paasnaam-argocd` was created;
        2. Assess that the `quota_label` label was used as selector on the quota;
        3. Assess that the quota selector was set in such a manner so that only the `paasnaam-sso` namespace is selected;
        4. Assess that the size of the quota equals the size of the default quota specified in the paas_config;
    5. Assess default_permissions;
        1. Rolebindings to service account etc. (TODO: can these RBs be created without the existence of a ServiceAccount?)

Post scenarios: reset environment to clean slate.

## Capability Tekton

Check Quota


## Capability Grafana

Check Quota


## Capability SSO

What we test: creating a `Paas` with the `SSO` capability enabled.

Scenarios:

1. Create a minimal `Paas` SSO capability enabled, no capability quota;
    1. Assess that the list entry exists in the applicationset;
    2. Assess that the namespace: `paasnaam-sso` was created;
    3. Assess quota;
        1. Assess that a quota with the name `paasnaam-sso` was created;
        2. Assess that the `quota_label` label was used as selector on the quota;
        3. Assess that the quota selector was set in such a manner so that only the `paasnaam-sso` namespace is selected;
        4. Assess that the size of the quota equals the size of the default quota specified in the paas_config;
2. Remove the `Paas` from step 1;
    1. Assess that the `Quota` was removed;
    2. Assess that the `Namespace` was removed;
    3. Assess that the list entry in the `ApplicationSet` was removed;

Post scenarios: reset environment to clean slate.