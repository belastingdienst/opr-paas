# API Reference

## Packages
- [cpet.belastingdienst.nl/v1alpha1](#cpetbelastingdienstnlv1alpha1)


## cpet.belastingdienst.nl/v1alpha1

Package v1alpha1 contains API Schema definitions for the  v1alpha1 API group

### Resource Types
- [Paas](#paas)
- [PaasConfig](#paasconfig)
- [PaasConfigList](#paasconfiglist)
- [PaasList](#paaslist)
- [PaasNS](#paasns)
- [PaasNSList](#paasnslist)



#### ConfigArgoPermissions



Deprecated: ArgoCD specific code will be removed from the operator



_Appears in:_
- [PaasConfigSpec](#paasconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `default_policy` _string_ | Deprecated: ArgoCD specific code will be removed from the operator<br />The optional default policy which is set in the ArgoCD instance |  | Optional: \{\} <br /> |
| `resource_name` _string_ | Deprecated: ArgoCD specific code will be removed from the operator<br />The name of the ArgoCD instance to apply ArgoPermissions to |  | Required: \{\} <br /> |
| `role` _string_ | Deprecated: ArgoCD specific code will be removed from the operator<br />The name of the role to add to Groups set in ArgoPermissions |  | Required: \{\} <br /> |
| `header` _string_ | Deprecated: ArgoCD specific code will be removed from the operator<br />The header value to set in ArgoPermissions |  | Required: \{\} <br /> |


#### ConfigCapPerm

_Underlying type:_ _object_





_Appears in:_
- [ConfigCapability](#configcapability)



#### ConfigCapabilities

_Underlying type:_ _[map[string]ConfigCapability](#map[string]configcapability)_





_Appears in:_
- [PaasConfigSpec](#paasconfigspec)



#### ConfigCapability







_Appears in:_
- [ConfigCapabilities](#configcapabilities)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `applicationset` _string_ | Name of the ArgoCD ApplicationSet which manages this capability |  | Required: \{\} <br /> |
| `quotas` _[ConfigQuotaSettings](#configquotasettings)_ | Quota settings for this capability |  | Required: \{\} <br /> |
| `extra_permissions` _[ConfigCapPerm](#configcapperm)_ | Extra permissions set for this capability |  | Required: \{\} <br /> |
| `default_permissions` _[ConfigCapPerm](#configcapperm)_ | Default permissions set for this capability |  | Required: \{\} <br /> |


#### ConfigLdap







_Appears in:_
- [PaasConfigSpec](#paasconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `host` _string_ | LDAP server hostname |  | Required: \{\} <br /> |
| `port` _integer_ | LDAP server port |  | Required: \{\} <br /> |


#### ConfigQuotaSettings







_Appears in:_
- [ConfigCapability](#configcapability)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `clusterwide` _boolean_ | Is this a clusterwide quota or not |  | Required: \{\} <br /> |
| `ratio` _integer_ | The ratio of the requested quota which will be applied to the total quota |  | Required: \{\} <br /> |
| `defaults` _object (keys:[ResourceName](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#resourcename-v1-core), values:[Quantity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#quantity-resource-api))_ | The default quota which the enabled capability gets |  | Required: \{\} <br /> |
| `min` _object (keys:[ResourceName](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#resourcename-v1-core), values:[Quantity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#quantity-resource-api))_ | The minimum quota which the enabled capability gets |  | Required: \{\} <br /> |
| `max` _object (keys:[ResourceName](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#resourcename-v1-core), values:[Quantity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#quantity-resource-api))_ | The maximum quota which the capability gets |  | Required: \{\} <br /> |


#### ConfigRoleMappings

_Underlying type:_ _object_





_Appears in:_
- [PaasConfigSpec](#paasconfigspec)





#### NamespacedName







_Appears in:_
- [PaasConfigSpec](#paasconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  | Required: \{\} <br /> |
| `namespace` _string_ |  |  | Required: \{\} <br /> |


#### Paas



Paas is the Schema for the paas API



_Appears in:_
- [PaasList](#paaslist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `cpet.belastingdienst.nl/v1alpha1` | | |
| `kind` _string_ | `Paas` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[PaasSpec](#paasspec)_ |  |  |  |


#### PaasCapabilities

_Underlying type:_ _[map[string]PaasCapability](#map[string]paascapability)_





_Appears in:_
- [PaasSpec](#paasspec)



#### PaasCapability







_Appears in:_
- [PaasCapabilities](#paascapabilities)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Do we want to use this capability, default false |  |  |
| `gitUrl` _string_ | The URL that contains the Applications / Application Sets to be used by this capability |  |  |
| `gitRevision` _string_ | The revision of the git repo that contains the Applications / Application Sets to be used by this capability |  |  |
| `gitPath` _string_ | the path in the git repo that contains the Applications / Application Sets to be used by this capability |  |  |
| `quota` _[Quota](#quota)_ | This project has it's own ClusterResourceQuota settings |  |  |
| `sshSecrets` _object (keys:string, values:string)_ | You can add ssh keys (which is a type of secret) for capability to use for access to bitBucket<br />They must be encrypted with the public key corresponding to the private key deployed together with the Paas operator |  |  |
| `extra_permissions` _boolean_ | You can enable extra permissions for the service accounts beloning to this capability<br />Exact definitions is configured in Paas Configmap<br />Note that we want to remove (some of) these permissions in future releases (like self-provisioner) |  |  |


#### PaasConfig







_Appears in:_
- [PaasConfigList](#paasconfiglist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `cpet.belastingdienst.nl/v1alpha1` | | |
| `kind` _string_ | `PaasConfig` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[PaasConfigSpec](#paasconfigspec)_ |  |  |  |


#### PaasConfigList



PaasConfigList contains a list of PaasConfig





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `cpet.belastingdienst.nl/v1alpha1` | | |
| `kind` _string_ | `PaasConfigList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[PaasConfig](#paasconfig) array_ |  |  |  |


#### PaasConfigSpec







_Appears in:_
- [PaasConfig](#paasconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `decryptKeyPaths` _string array_ | Deprecated: Will be replaced by a secretRef to overcome caching<br />Paths where the manager can find the decryptKeys to decrypt Paas'es |  | MinItems: 1 <br />Required: \{\} <br /> |
| `debug` _boolean_ | Enable debug information generation or not | false | Optional: \{\} <br /> |
| `capabilities` _[ConfigCapabilities](#configcapabilities)_ | A map with zero or more ConfigCapability |  | Required: \{\} <br /> |
| `whitelist` _[NamespacedName](#namespacedname)_ | Deprecated: Whitelist code will be removed from the operator to make it more generic<br />A reference to a configmap containing a whitelist of LDAP groups to be synced using LDAP sync |  | Required: \{\} <br /> |
| `ldap` _[ConfigLdap](#configldap)_ | LDAP configuration for the operator to add to Groups |  | Optional: \{\} <br /> |
| `argopermissions` _[ConfigArgoPermissions](#configargopermissions)_ | Deprecated: ArgoCD specific code will be removed from the operator<br />Permissions to set for ArgoCD instance |  | Required: \{\} <br /> |
| `applicationset_namespace` _string_ | Namespace in which ArgoCD applicationSets will be found for managing capabilities | argocd | Required: \{\} <br /> |
| `quota_label` _string_ | Label which is added to clusterquotas | clusterquotagroup | Optional: \{\} <br /> |
| `requestor_label` _string_ | Name of the label used to define who is the contact for this resource | requestor | Optional: \{\} <br /> |
| `managed_by_label` _string_ | Name of the label used to define by whom the resource is managed. | argocd.argoproj.io/managed-by | Optional: \{\} <br /> |
| `exclude_appset_name` _string_ | Deprecated: ArgoCD specific code will be removed from the operator<br />Name of an ApplicationSet to be set as ignored in the ArgoCD bootstrap Application |  | Required: \{\} <br /> |
| `rolemappings` _[ConfigRoleMappings](#configrolemappings)_ | Grant permissions to all groups according to config in configmap and role selected per group in paas. |  | Optional: \{\} <br /> |




#### PaasGroup







_Appears in:_
- [PaasGroups](#paasgroups)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `query` _string_ |  |  |  |
| `users` _string array_ |  |  |  |
| `roles` _string array_ |  |  |  |


#### PaasGroups

_Underlying type:_ _[map[string]PaasGroup](#map[string]paasgroup)_





_Appears in:_
- [PaasSpec](#paasspec)



#### PaasList



PaasList contains a list of Paas





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `cpet.belastingdienst.nl/v1alpha1` | | |
| `kind` _string_ | `PaasList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Paas](#paas) array_ |  |  |  |


#### PaasNS



PaasNS is the Schema for the paasns API



_Appears in:_
- [PaasNSList](#paasnslist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `cpet.belastingdienst.nl/v1alpha1` | | |
| `kind` _string_ | `PaasNS` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[PaasNSSpec](#paasnsspec)_ |  |  |  |


#### PaasNSList



PaasNSList contains a list of PaasNS





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `cpet.belastingdienst.nl/v1alpha1` | | |
| `kind` _string_ | `PaasNSList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[PaasNS](#paasns) array_ |  |  |  |


#### PaasNSSpec



PaasNSSpec defines the desired state of PaasNS



_Appears in:_
- [PaasNS](#paasns)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `paas` _string_ | Foo is an example field of PaasNS. Edit paasns_types.go to remove/update |  |  |
| `groups` _string array_ |  |  |  |
| `sshSecrets` _object (keys:string, values:string)_ |  |  |  |




#### PaasSpec



PaasSpec defines the desired state of Paas



_Appears in:_
- [Paas](#paas)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `capabilities` _[PaasCapabilities](#paascapabilities)_ | Capabilities is a subset of capabilities that will be available in this Paas Project |  |  |
| `requestor` _string_ | Requestor is an informational field which decides on the requestor (also application responable) |  |  |
| `groups` _[PaasGroups](#paasgroups)_ |  |  |  |
| `quota` _[Quota](#quota)_ | Quota defines the quotas which should be set on the cluster resource quota as used by this Paas project |  |  |
| `namespaces` _string array_ | Namespaces can be used to define extra namespaces to be created as part of this Paas project |  |  |
| `sshSecrets` _object (keys:string, values:string)_ | You can add ssh keys (which is a type of secret) for ArgoCD to use for access to bitBucket<br />They must be encrypted with the public key corresponding to the private key deployed together with the Paas operator |  |  |
| `managedByPaas` _string_ | Indicated by which 3rd party Paas's ArgoCD this Paas is managed |  |  |








