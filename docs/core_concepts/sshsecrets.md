# sshSecrets

SshSecrets is implemented to enable bootstrappign a project 100% 'as code'.
The idea is that people can create a PaaS to bootstrap an application fornt to back, including required namespaces, quotas, a application specific ArgoCD and a running application, in one go.
But (when using private repositories) ArgoCD needs to be provided with a ssh key (as a secret) for ArgoCD to gain access to the git repository.
These ssh secrets need to be provided before ArgoCD can start acting on repo contents, which is why providing these secrets is part of the PaaS solution.

Other consideration was that we want sshSecrets to be defined in a PaaS, PaaS can be readable to the world, and we don't want the secrets to be open, which is why we implemented encryption.
Encryption is based on rsa where a prublic key (shared with everyone) is used to encrypt, and a private key (deployed with the PaaS operator) is used to decrypt.
Which that everyone can encrypt, but only the PaaS operator can decrypt.

**Note** that we implemented sshSecrets for this usecase, but they are implemented generic, and can also be used to seed secrets into other namaspaces (capability and user namespaces)...

For ease, and to enable extra management capabilities, the PaaS operator comes with additional tooling:

- an API, which can be used to encrypt without needing to share the public key.
- a crypt tool which can be leveraged to encrypt, reencrypt, generate key pairs, and inspect encrypted keys
  Both of these tools require access to the private key to be usable...

## How it works

- A DevOps engineer generates a ssh key pair
- The DevOps engineer configures his public ssh key to be accepted by his git repository (e.a. github, gitlab, gitea, bitbucket, etc.)
- The DevOps engineer encrypts the private ssh key with the api, or with the cli.
  Encryption is done using the PaaS public key (the result can only be decrypted using the PaaS private key).
  The result is called a `sshSecret`
- The DevOps engineer creates (or modifies) a PaaS with the sshSecret
- PaaS controller creates a PaasNs with the `sshSecret` included
- PaasNs controller creates the required namespaces, ArgoCD resource and ArgoCD repo definition (which is a k8s secret)
- ArgoCD contacts git and uses the secret to authenticate
- ArgoCD creates resources as is defined in the git repository
- Application comes alive

```kroki-blockdiag
blockdiag {
  "ssh private key" -> encryption -> sshSecret -> PaaS -> operator -> "ArgoCD namespace";
  "PaaS public key" -> encryption;
  "PaaS private key" -> operator;
  operator -> "ArgoCD quota";
  operator -> "ArgoCD";
  operator -> "ArgoCD repo (secret)";
  operator -> "Other capabilities";
  operator -> "Other namespaces";
  operator -> "...";
  "ssh private key" [color = "greenyellow"];
  "PaaS public key" [color = "pink"];
  "PaaS private key" [color = "pink"];
  "ArgoCD repo (secret)" [color = orange];
}
```

## Defining an sshSecret

sshSecrets are processed by the PaasNs controller and as such need to be defined in the PaasNs.
Additionally, sshSecrets can also be created in a PaaS.

### Defining sshSecrets in a PaaS

The PaaS controller only manages PaasNs's created by the conttoller as defined by the PaaS.
**Note** PaasNs resources which are not created by the PaaS controller require sshSecrets to be configured as part of the PaasNs definition.

SshSecrets can be defined in a PaaS on 2 levels:

- as part of the spec, in which case the PaaS controller will add the sshSecret to every PaasNs created by the PaaS controller
- as part of capability, in which case the PaaS controller will add the sshSecret to PaasNs created for this capability specifically
  This is the normal usecase (part of the argocd capability)

Example:

```yaml
---
apiVersion: cpet.belastingdienst.nl/v1alpha1
kind: Paas
metadata:
  name: my-paas
spec:
  # Specifying a sshSecret for all capability- and functional- namespaces
  sshSecrets:
    "ssh://git@github.com/belastingdienst/paas.git": >-
      2wkeKebCnqgl...L/jDAUmhWG3ng==
  capabilities:
    argocd:
      # Specifying a sshSecret for a specific capability namespace
      sshSecrets:
        "ssh://git@github.com/belastingdienst/paas.git": >-
          2wkeKebCnqgl...L/jDAUmhWG3ng==
  requestor: my-team
  quota:
    limits.cpu: "40"
```

### Defining sshSecrets in a PaasNs

The PaasNs controller is the one to manage the secrets in the PaaS namespaces as defined in the PaasNs (either manually created or managed by the PaaS controller).
The PaasNs controller will update ssh secrets in the namespace if the sshSecret value is changed in the PaasNs resource.
But when the key changes (e.a. `ssh://git@github.com/belastingdienst/paas.git` in the example below), the original ssh secret is not removed.

```yaml
---
apiVersion: cpet.belastingdienst.nl/v1alpha1
kind: PaasNs
metadata:
  name: my-ns
  namespace: my-paas-argocd
spec:
  paas: my-paas
  sshSecrets:
    "ssh://git@github.com/belastingdienst/paas.git": >-
      2wkeKebCnqgl...L/jDAUmhWG3ng==
```
