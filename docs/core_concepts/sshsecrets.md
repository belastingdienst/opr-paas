# sshSecrets

SshSecrets is implemented to enable bootstrapping a project 100% 'as code'.

The idea is that people can create a Paas to bootstrap an application front to
back, including required namespaces, quotas, a application specific ArgoCD and a
running application, in one go.

However, when using private repositories, ArgoCD needs to be provided with an SSH
key (as a secret) for ArgoCD to gain access to the git repository. These ssh secrets
need to be provided before ArgoCD can start acting on repo contents, which is why
providing these secrets is part of the Paas solution.

Another consideration was that we want `sshSecrets` to be defined in a Paas, and
since Paas can be readable to the world, and we naturally don't want the secrets
to be open, which is why we implemented encryption.

Encryption is based on RSA where a public key (shared with everyone) is used to
encrypt, and a private key (deployed with the Paas operator) is used to decrypt.
Which that everyone can encrypt, but only the Paas operator can decrypt.

!!! Note
    Note that we implemented `sshSecrets` for this use case, but they are implemented
    generically, and can also be used to seed secrets into other namespaces (capability
    and user namespaces)...

For ease of use, and to enable extra management capabilities, the Paas operator
comes with additional tooling:

- an API, which can be used to encrypt without needing to share the public key;
- a crypt tool which can be leveraged to encrypt, re-encrypt, generate key pairs,
  and inspect encrypted keys;

Both of these tools require access to the private key to be usable...

## How it works

- A DevOps engineer generates a SSH key pair;
- The DevOps engineer configures his public SSH key to be accepted by his git
  repository (e.a. github, gitlab, gitea, bitbucket, etc.)
- The DevOps engineer encrypts the private SSH key with the api, or with the CLI;
  Encryption is done using the Paas public key (the result can only be decrypted
  using the Paas private key).
  The result is called a `sshSecret`.
- The DevOps engineer creates (or modifies) a Paas with the `sshSecret`;
- Paas controller creates a PaasNs with the `sshSecret` included;
- PaasNs controller creates the required namespaces, ArgoCD resource and ArgoCD
  repo definition (which is a K8S secret);
- ArgoCD contacts git and uses the secret to authenticate;
- ArgoCD creates resources as is defined in the git repository;
- Application comes alive;

```kroki-blockdiag
blockdiag {
  "SSH private key" -> encryption -> sshSecret -> Paas -> operator -> "ArgoCD namespace";
  "Paas public key" -> encryption;
  "Paas private key" -> operator;
  operator -> "ArgoCD quota";
  operator -> "ArgoCD";
  operator -> "ArgoCD repo (secret)";
  operator -> "Other capabilities";
  operator -> "Other namespaces";
  operator -> "...";
  "SSH private key" [color = "greenyellow"];
  "Paas public key" [color = "pink"];
  "Paas private key" [color = "pink"];
  "ArgoCD repo (secret)" [color = orange];
}
```

## Defining an `sshSecret`

`sshSecret`s are processed by the PaasNs controller and as such need to be defined
in the PaasNs. Additionally, `sshSecret`s can also be created in a Paas.

### Defining `sshSecret`s in a Paas

The Paas controller only manages PaasNs's created by the controller as defined
by the Paas.

!!! Note
    PaasNs resources which are not created by the Paas controller require `sshSecrets`
    to be configured as part of the PaasNs definition.

`sshSecret`s can be defined in a Paas on 2 levels:

- as part of the spec, in which case the Paas controller will add the `sshSecret`
  to every PaasNs created by the Paas controller
- as part of capability, in which case the Paas controller will add the `sshSecret`
  to PaasNs created for this capability specifically

  This is the normal use case (part of the argocd capability)

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

### Defining `sshSecret`s in a PaasNs

The PaasNs controller is the one to manage the secrets in the Paas namespaces a
defined in the PaasNs (either manually created or managed by the Paas controller).

The PaasNs controller will update SSH secrets in the namespace if the `sshSecret`
value is changed in the PaasNs resource. However, when the key changes
(e.a. `ssh://git@github.com/belastingdienst/paas.git` in the example below), the
original SSH secret is not removed.

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
