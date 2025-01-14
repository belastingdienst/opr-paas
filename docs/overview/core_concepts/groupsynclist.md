# Caas-groupsynclist

The current implementation we have running on Openshift internally relies on
running `oc adm groups sync` commands periodically.

The information for this job comes from a ConfigMap called `groupsynclist`.

The Paas operator will manipulate the data in a key of a configured ConfigMap. The targeted configmap is 
configured through the `paasconfig.spec.groupsynclist`. The keyname can be configured with `paasconfig.spec.groupsynclistkey`.

We are in the process of changing this solution to a more K8S generic solution for
management of Users and Groups.

If you are working on ldap integration in a more K8S generic way, feel free to
issue a PR.
