# E2E test cases for opr-paas

Below you will find a list of features that we want to test using end-to-end testing.
For each feature, the setup and the assessments are listed.

TODO:
1. Take up `RoleMappings` in `testConfig`;
2. Test `extra_permissions` / `default_permissions` in config;

## PaasNs

1. Create `PaasNs` while the `Paas` mentioned in the spec, does not exist;
    * Assess that the namespace is not created;
    * Assess that the status of PaasNs contains the correct error;
2. Create a minimal, missing Paas that is referenced in the PaasNs from step 1;
    1. Remove the PaasNs from 1;
    2. Create the same PaasNs from 1 again;
    3. Assess that the namespace exists;
    4. Assess that the namespace has the `quota_label` label with the value of the .. work out more;
    5. Assess that the status of the `PaasNs` no longer contains any errors;
3. Remove the `PaasNs`;
    1. Assess that the namespace was removed;
4. Cleanup, remove the Paas;

## ClusterResourceQuota

Extra info: the spec.quota is not clusterWide hence separate test.

1. Create a minimal Paas. Enter a quota for spec.quota;
    1. Assess that a quota with the name of the PaaS was created;
    2. Assess that the quota_label was used as selector on the quota;
    3. Assess that the size of the quota equals the size as specified in the spec.quota;
2. Update the Paas, adjust the spec.quota;
    1. Assess that the size of the quota was adjusted and now equals the size as specified in the spec.quota;
3. Remove the Paas;
    1. Assess that the quota with the name of the PaaS was removed;

## Namespaces

1. Create a minimal Paas without namespaces;
    1. Assess that 1 namespace was created with the name of the Paas;
    2. Assess that no PaasNs's exist that are linked to this Paas;
2. Update the Paas and add 2 namespaces to the spec;
    1. Assess that 1 namespace was created with the name of the Paas;
    2. Assess that 2 PaasNs's exist, in the namespace of the PaaS, named according to the spec.namespaces entries;
3. Update the Paas, remove the namespaces;
    1. Assess that the PaasNs's were removed from the Paas namespace;
    2. Assess that the 'paas' namespace was removed;

## Groups Users

1. Create a minimal Paas with a single namespace and a group with a single user, without a specified role;
    1. Assess that a Group was created with the correct name;
    2. Assess that the user is a member of the group;
    3. Assess that the correct labels are placed on the group;
    4. Assess Owner Reference naar Group naar correcte Paas  !!!!!!
    5. Assess there are no changed on Whitelist;
    6. Assess that the rolebinding on the namespace points to the group to the default role;
    7. Assess that the rolebinding is not applied to the default Paas namespace (name of the Paas)
2. Update the Paas from step 1, add a group with a specific role, other than default (see test_config), add different user than in step 1;
    *Assess everything one more time*
    1.  Assess that a Group was created with the correct name;
    2. Assess that the user is a member of the group;
    3. Assess that the correct labels are placed on the group;
    4. Assess Owner Reference naar Group naar correcte Paas  !!!!!!
    5. Assess there are no changed on Whitelist;
    6. Assess that the rolebinding on the namespace points to the group to the specified role;
    7. Assess that the rolebinding is not applied to the default Paas namespace (name of the Paas);
3. Remove the Paas;
   *There are known issues on groups, updating / removing does not go perfectly. Goal here is to test the baseline in the current situation*
   *Determine what that baseline currently is.*

## Groups Query

1. Create a minimal Paas with one namespace and a group with a Query without having a Role specified;
    1. Assess that a Group with the correct name was created;
    2. Assess there are no users in the group;
    3. Assess the correct labels were added onto the group;
    4. Assess Owner Reference naar Group naar correcte Paas    !!!!!!!
    5. Assess the query was added to Whitelist;
    6. Assess that the rolebinding on the namespace points to the group to the specified role;
    7. Assess that the rolebinding is not applied to the default Paas namespace (name of the Paas);
2. Update Paas from step 1, add group with a specific role, other than default (see test_config), add another query than was added for step 1;
    *Assess everything one more time*
    1. Assess that a Group with the correct name was created;
    2. Assess there are no users in the group;
    3. Assess the correct labels were added onto the group; (no ldap things)
    4. Assess Owner Reference naar Group naar correcte Paas   !!!!!!!
    5. Assess changes were made to Whitelist;
    6. Assess that the rolebinding on the namespace points to the group to the specified role;
    7. Assess that the rolebinding is not applied to the default Paas namespace (name of the Paas);
3. Remove the Paas;
   *There are known issues on groups, updating / removing does not go perfectly. Goal here is to test the baseline in the current situation*
   *Determine what that baseline currently is.*


// TODO, there a known issue regarding groups. Good regression test to model the following steps:
1. Create Paas with query;
2. Update Paas from step 1, remove query and add users to the group;
3. Group is not removed from whitelist;
4. Sync fails because the required `ldap.uid` doesn't match the groupname. (We don't fully test sync.)

## Secrets

*The capabilities are also used to test clusterwide quotas*

## Capability ArgoCD

1. Create a minimal Paas with ArgoCD capability enabled;
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

## Capability Tekton

Check Quota


## Capability Grafana

Check Quota


## Capability SSO

1. Create a minimal Paas SSO capability enabled, no capability quota;
    1. Assess that the list entry exists in the applicationset;
    2. Assess that the namespace: `paasnaam-sso` was created;
    3. Assess quota;
        1. Assess that a quota with the name `paasnaam-sso` was created;
        2. Assess that the `quota_label` label was used as selector on the quota;
        3. Assess that the quota selector was set in such a manner so that only the `paasnaam-sso` namespace is selected;
        4. Assess that the size of the quota equals the size of the default quota specified in the paas_config;
2. Remove the Paas from step 1;
    1. Assess that the Quota was removed;
    2. Assess that the Namespace was removed;
    3. Assess that the list entry in the applicationset was removed;