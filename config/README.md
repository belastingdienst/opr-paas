# IMPORTANT
This folder is here as an example only.
The actual argocd config is actually maintained in opr-paas-config
When changing the crd, first run `make manifests` in the root of this repo.
Then copy config/crd/bases/cpet.belastingdienst.nl_paas.yaml to the opr-paas-config repo and distribute with ArgoCD from there
