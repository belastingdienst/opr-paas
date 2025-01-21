---
title: Managing ssh Secrets
summary: How ssh secrets can be leveraged to create secrets in paas namespaces in a secure manner.
authors:
  - hikarukin
  - devotional-phoenix-97
date: 2025-01-20
---

# SSH Secrets

It is possible to define secrets to be created in a namespace for a specific capability,
or in every namespace generically. The main use case is to create ssh secrets in the
ArgoCD namespace so that it can read private repositories, which is where the name
sshSecrets came from in the first place. However, they can be used with other capabilities,
and/or application namespaces as well.

More info can be found in [Core Concepts documentation on ssh Secrets](../overview/core_concepts/sshsecrets.md).

## Prerequisites

SshSecrets are encrypted using asymmetric encryption and therefore require a public
and private keypair. Keypairs must be generated and managed by administrators and can
provide the public key to Users for encrypting sshSecrets. For more info, please see
the [Admin guide on configuring ssh secret encryption](../administrators-guide/sshSecrets.md).

## Encrypting secrets

You can download the crypttool from the [Downloads section of the latest release](https://github.com/belastingdienst/opr-paas/releases).
Once downloaded, the crypttool has two options for encrypting content:

### Encrypting a file with crypttool

!!! example

    ```bash
    ./crypttool --encrypt-from-file ~/.ssh/id_rsa -paas-name tst-tst -key ~/Downloads/public.bin
    ```

### Encrypting from stdin with crypttool

!!! example

    ```bash
    echo -e 'private investigations' | ./crypttool --encrypt-from-stdin -paas-name tst-tst -key ~/Downloads/public.bin
    ```

### using cURL against the webservice api

!!! example

    ```bash
    ENDPOINT_URL="https://paas-webservice-paas-system.apps.mycluster.example/v1/encrypt"
    JSONTYPE='Content-Type: application/json'
    PAAS=tst-tst
    SECRET=$(awk '{printf "%s\\n", $0}' ~/.ssh/id_rsa)
    curl -X POST "${ENDPOINT_URL}" -H "${JSONTYPE}" -d '{"paas":"'${PAAS}'","secret":"'${SECRET}'"}'
    ```

### other options

Options are endless. Be creative...

## defining sshSecrets

Encrypted SSH Secrets can be specified in multiple places.

By defining the secret in in the Paas spec directly (`Paas.spec.sshSecrets`) the
secret will be created in all namespaces belonging to the paas.

!!! example

    Setting an sshSecret for all namespaces

    ```yaml
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: Paas
    metadata:
      name: tst-tst
    spec:
      sshSecrets:
        'ssh://git@my-git-host/my-git-repo.git': >-
          2wkeKe...g==
    ```

By defining the secret as part of a capability (such as `argocd`), the secret will
be deployed in the namespace belonging to the capability specifically.

!!! example

    Setting an sshSecret for a specific capability

    ```yaml
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: Paas
    metadata:
      name: tst-tst
    spec:
      capabilities:
        argocd:
          ...
          sshSecrets:
            'ssh://git@my-git-host/my-git-repo.git': >-
              2wkeKe...g==
    ```

By defining the secret as part of a PaasNs, the secret will be deployed in the
corresponding namespace only.

!!! example

    ```yaml
    ---
    apiVersion: cpet.belastingdienst.nl/v1alpha1
    kind: Paas
    metadata:
      name: tst-tst
    spec:
      capabilities:
        # The argocd capability enabled
        argocd:
          enabled: true
      requestor: my-team
      quota:
        limits.cpu: "40"
      sshSecrets:
        'ssh://git@my-git-host/my-git-repo.git': >-
          2wkeKe...g==
    ```
