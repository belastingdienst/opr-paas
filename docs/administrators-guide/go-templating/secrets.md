---
title: Go Template secret options
summary: A detailed description of how Go Template is used to define the format for secrets
authors:
  - Devotional Phoenix
date: 2025-06-18
---

# Secrets with go-templating

The original secret implementation was specifically designed for generating ArgoCD repo secrets, but the secrets were less useful for other capabilities, namespaces, etc.
See [issue 1056](https://github.com/belastingdienst/opr-paas/issues/1056) for more info.

We have now replaced the implementation with a go-templating driven alternative, with allows administrators to define the exact format for secrets per capability, as well as for all namespaces specifically.
In the future we expect to expand the solution to
- also support External Secrets integration
- replace the default template (which currently defaults to the argocd behavior which has very limitted use outside of argocd

## How it works

In original solution a specific secret format was decided and applied for all secrets created by the operator.
In the new solution, Administrators can define go templates in the PaasConfig to define the format of secrets.
The template can be set in PaasConfig.spec.namespace_secrets, which will define the format for all secrets in namespaces defined in a Paas.spec.namespaces, and/or as a PaasNs.
Templates can also be set in PaasConfig.spec.capabilities[].secrets, in which case it will determine the format for a specific capability.

## Format

The templates can use info from the PaasConfig, and from the Paas. They can use special functions (described in [below chapter Special functions](#Special_functions)).
The template should result in a yaml string containing a map of maps (map[string]map[string]string{}).
In this result, the key for the first level defines the name of the secret to be created. The key for the second level defines the key in .data to be set to the value.

So, if the template resolves into
!!! example
    ```yaml
    my-secret: 
      my-key-1: some secret value
      my-key-2: some other secret value
    my-other-secret:
      other-key: other secret
    ```

The paas operator would create 2 secrets:
- one called `my-secret`, having 2 key/value pairs:
  - `my-key-1` which is set to the base64 version of the string `some secret value`
  - `my-key-2`, which is set to the base64 version of the string `some other secret value`
- another one called `my-other-secret` having only one pair:
  - `other key`, which is set to the base64 version of the string `other secret`

## If you don't set a template

For now we have added a default, which results in the same behavior as before this implementation was introduced:

!!! example
    ```
    {{- $secrets := dict -}}
    {{- range $key, $value := getPaasSecrets -}}
      {{- $hash := $key | sha512Sum | trunc 8 -}}
      {{- $secretName := print "paas-ssh-" $hash -}}
      {{- $secretData := dict "type" "git" "url" $key "sshPrivateKey" $value -}}
      {{- $_ := $secrets | set $secretName $secretData -}}
    {{- end -}}
    {{- $secrets | toYAML -}}
    ```
!!! note
    In the next major version we plan to require a template to be specified, and will then remove this default.

## Special functions

As you can see in the examples, we have added a function called getPaasSecrets.
This function parses all secrets that apply for this namespace, which could be
- For a capability: capability secrets and paas secrets
- For a non-capability namespace (Paas.spec.namespaces, or a PaasNs): namespace secrets and paas secrets.

Next to getPaasSecrets, we have also added getPaasSecret (without the s).
This function can be used to retrieve the (decrpyted) value of a specific secret.
Example usage: `{{ mySecret := getPaasSecret "my-secret" }}`

## Adviced template for ArgoCD

Below template can be used for the ArgoCD capability:

!!! example
    ```
    {{- $secrets := dict -}}
    {{- range $key, $value := getPaasSecrets -}}
      {{- $hash := $key | sha512Sum | trunc 8 -}}
      {{- $secretName := print "paas-ssh-" $hash -}}
      {{- $secretData := dict "type" "git" "url" $key "sshPrivateKey" $value -}}
      {{- $_ := $secrets | set $secretName $secretData -}}
    {{- end -}}
    {{- $secrets | toYAML -}}
    ```

This results in a secret for every paas secret, where the name of the paas secret is used as the url, and the value as the actual secret.
The name of the secret is derived from the hashed value of the name of the paas secret.

## Adviced template for Tekton

Below template can be used for the Tekton capability:

!!! example
    ```
    {{- $auths := dict -}}
    {{- range $name, $decrypted := getPaasSecrets -}}
      {{- $auth := base64Encode $decrypted -}}
      {{- $parts := split ":" $decrypted -}}
      {{- $username := index $parts "_0" -}}
      {{- $password := index $parts "_1" -}}
      {{- $authData := dict "username" $username "password" $password "auth" $auth -}}
      {{- $_ := $auths | set $name $authData -}}
    {{- end -}}
    {{- $dockerConfigJSON := toJSON (dict "auths" $auths) -}}
    {{- $secrets := dict "container-repo-token" (dict ".dockerconfigjson" $dockerConfigJSON) -}}
    {{ $secrets | toYAML }}
    ```

This results in one secret called container-repo-token, with one value called .dockerconfigjson.
All PaasSecrets are added as a auth token to the json value in `.dockerconfigjson`.

## Adviced template for other capabilities and namespaces

For all other situations you could fall back to this template:

!!! example
    ```
    {{- $scrt := getPaasSecrets -}}
    {{- if gt (len $scrt) 0 -}}
      {{- $result := dict "paas-secrets" $scrt -}}
      {{- $result | toYAML -}}
    {{- end -}}
    ```

This results in one secret, with all paas-secrets added as key/value pairs.
This will also be the future default when no template is set.

If you need another go-template, feel free to request support by issuing a ticket.
