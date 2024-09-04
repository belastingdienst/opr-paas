# Caas-whitelist

The current implementation we have internally implemented relies on running
`oc adm groups sync` commands periodically.

The information for this job comes from a ConfigMap called `caas_whitelist`.

We are in the making of changing this solution to a more K8S generic solution for
management of Users and Groups.

If you are working on ldap integration in a more K8S generic way, feel free to issue a PR.
