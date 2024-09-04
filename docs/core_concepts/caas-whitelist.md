# Caas-whitelist

The current implementation we have internally implemented relies on running `oc adm groups sync` commands periodically.
The information for this job comes from a configmap called `caas_whitelist`.
We are in the making of changing this solution to a more K8s generic solution for management of Users and Groups.
If you are working on ldap integration in a more k8s generic way, feel free to issue a PR.
