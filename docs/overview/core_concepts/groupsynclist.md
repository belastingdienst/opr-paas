# Caas-groupsynclist

The current deployment of Paas, we have running on Openshift internally, relies on running `oc adm groups sync` periodically.

The information for this sync comes from a ConfigMap called `groupsynclist`.

Only the LDAP queries mentioned in this ConfigMap are synced to OpenShift. We have a job in place, to keep this ConfigMap up-to-date with all Paas.groups.query field values on the cluster.

We are in the process of changing this logic to a more K8S generic solution for
management of Users and Groups.

If you are working on ldap integration in a more K8S generic way, feel free to
issue a PR.
