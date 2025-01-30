---
title: Configuring ssh Secret encryption
summary: A detailed description of requirements for setting up the ssh Secret encryption feature
authors:
  - Devotional Phoenix
date: 2025-01-20
---

# Ssh secret encryption

The Paas operator includes features to manage secrets in namespaces of a Paas.
The main use case is to create ssh secrets in the ArgoCD namespace so that it can
read private repositories, which is where the name sshSecrets came from in the first
place. However, they can be used with other capabilities, and/or application namespaces too.

SshSecrets are encrypted using asymmetric encryption and therefore require a public
and private keypair. Keypairs must be generated, after which the Private Keys must
be added to the secret configured in the `PaasConfig.spec.privateKeySecret`, and
the public key must be provided to Users for encrypting the ssh Secrets (either directly,
or through the web service).

## Generating new secrets

New keys can be easily generated using the crypttool. You can download the crypttool
from the [Downloads section of the latest release](https://github.com/belastingdienst/opr-paas/releases).

Once downloaded, the crypttool can be used to generate a keypair as follows:

!!! example

    ```bash
    cd $(mktemp -d)
    crypttool generate --privateKeyFile private.bin --publicKeyFile public.bin
    ```

## Deploying new secrets

Once generated, the private key should be added to the secret configured in the `PaasConfig.spec.privateKeySecret`.

!!! note

    The secret as configured in the `PaasConfig.spec.privateKeySecret` can hold multiple keys.
    This feature is implemented so that key rotation (generating, deploying and reencryption)
    do not need to be performed instantly. The Paas operator tries to decrypt with all secrets
    and detects a successful decryption from one of the supplied keys.

For the next Paas reconciliation, the change is detected, and the new private key
will (also) be tried for decryption.

## Supplying new public key

### Directly

Once generated, the public key should be supplied to users that encrypt secrets.
They can be supplied directly, so that users can use the crypttool for encryption.
For more info, please refer to [user docs on ssh Secrets](../user-guide/02_ssh-secrets.md).

### Running the webservice

Another option is to run the webservice. To enable the webservice enable the webservice manifest:

!!! example

    ```bash
    cd $(mktemp -d)
    git clone https://github.com/belastingdienst/opr-paas.git
    cd opr-paas/manifests/default && \
    kustomize edit set image controller="ghcr.io/belastingdienst/opr-paas" && \
    kustomize edit set image webservice="ghcr.io/belastingdienst/webservice" && \
    kustomize edit add resource ../webservice && \
    kustomize build . | kubectl apply -f -
    ```

After that you can replace the publicKey data in the paas-sshsecrets-publickey ConfigMap,
k8s changes the mount and the webservice automatically picks up the file changes and uses the new key.

!!! warning

    When deploying the webservice application, you will be required to add an environment
    variable called `PAAS_WS_ALLOWED_ORIGINS` in which you either give `*` or one or more
    CORS related origins, comman separated.

    By default, the webservice will be deployed using `http://www.example.com` as a value
    and will not work for you.

### Reencryption

A proper encryption product also has options to cycle the encrypted data.
With the sshSecrets implementation in the operator, this is implemented with the crypttool.

Steps are:

- generate new keys
- deploy new keys
- for every paas:
  - check that original key still works
  - reencrypt with original private key and newly created public key
  - update reencrypted paas
- (optionally) remove original key

!!! note

    Reencryption requires the original private key which only admins should have access to.

!!! example

    ```bash
    set -o
    cd $(mktemp -d)
    kubectl get paas -o name | while read -r PAAS; do
      kubectl get paas "${PAAS}" > "${PAAS}.yaml"
      crypttool check-paas --privateKeyFiles ~/Downloads/oldpriv > "${PAAS}.pre.out"
      crypttool reencrypt --privateKeyFiles ~/Downloads/oldpriv --publicKeyFile ~/Downloads/newpublicKey "${PAAS}.yaml"
      crypttool check-paas --privateKeyFiles ~/Downloads/oldpriv > "${PAAS}.post.out"
      kubectl apply -f "${PAAS}.yaml"
    done
    ```
