package controllers

import (
	"context"
	"os"
	"sort"

	mydomainv1alpha1 "github.com/belastingdienst/opr-paas/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	DefaultCaasWhitelistNameSpace = "kube-system"
	DefaultCaasWhitelistName      = "caaswhitelist"
)

// CaasWhiteList returns a Namespaced object name which points to the
// Caas Whitelist where the ldap groupds should be defined
// Defaults point to kube-system.caaswhitelist, but can be overruled with
// the environment variables CAAS_WHITELIST_NAMESPACE and CAAS_WHITELIST_NAME
func CaasWhiteList() (wl types.NamespacedName) {
	if wl.Name = os.Getenv("CAAS_WHITELIST_NAME"); wl.Name == "" {
		wl.Name = DefaultCaasWhitelistName
	}
	if wl.Namespace = os.Getenv("CAAS_WHITELIST_NAMESPACE"); wl.Namespace == "" {
		wl.Namespace = DefaultCaasWhitelistNameSpace
	}
	return wl
}

func (r *PaasReconciler) ensureLdapGroupsConfigMap(
	whiteListConfigMap types.NamespacedName,
	groups Groups,
) error {
	// Create the ConfigMap
	wlConfigMap := CaasWhiteList()
	return r.Create(context.TODO(), &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      wlConfigMap.Name,
			Namespace: wlConfigMap.Namespace,
		},
		Data: map[string]string{
			"whitelist.txt": groups.AsString(),
		},
	})
}

// Simple struct to parse a string into a map of groups with key is cn so it will be unique
// struct can add and struct can be changed back into a string.
type Groups struct {
	by_key map[string]string
}

func NewGroups() *Groups {
	return &Groups{
		by_key: make(map[string]string),
	}
}

func (gs *Groups) DeleteByValue(byValue string) bool {
	for key, value := range gs.by_key {
		if value == byValue {
			delete(gs.by_key, key)
			return true
		}
	}
	return false
}

func (gs Groups) Len() int {
	return len(gs.by_key)
}

func (gs Groups) Add(other Groups) Groups {
	for key, value := range other.by_key {
		gs.by_key[key] = value
	}
	return gs
}

func (gs Groups) AddFromStrings(l []string) Groups {
	for _, group := range l {
		//CN=gkey,OU=org_unit,DC=example,DC=org
		if cn := strings.Split(group, ",")[0]; !strings.ContainsAny(cn, "=") {
			continue
		} else {
			key := strings.SplitN(cn, "=", 2)[1]
			gs.by_key[key] = group
		}
	}
	return gs
}

func (gs Groups) AddFromString(s string) Groups {
	return gs.AddFromStrings(strings.Split(s, "\n"))
}

func (gs Groups) AsString() string {
	values := make([]string, 0, len(gs.by_key))
	for _, v := range gs.by_key {
		values = append(values, v)
	}
	sort.Strings(values)
	return strings.Join(values, "\n")
}

// ensureLdapGroup ensures Group presence
func (r *PaasReconciler) EnsureLdapGroups(
	paas *mydomainv1alpha1.Paas,
) error {
	// See if group already exists and create if it doesn't
	cm := &corev1.ConfigMap{}
	wlConfigMap := CaasWhiteList()
	err := r.Get(context.TODO(), wlConfigMap, cm)
	groups := NewGroups().AddFromStrings(paas.Spec.Groups.LdapQueries())
	if err != nil && errors.IsNotFound(err) {
		// Create the ConfigMap
		return r.ensureLdapGroupsConfigMap(
			wlConfigMap,
			groups,
		)
	} else if err != nil {
		// Error that isn't due to the group not existing
		return err
	} else if whitelist, exists := cm.Data["whitelist.txt"]; !exists {
		cm.Data["whitelist.txt"] = groups.AsString()
	} else {
		configured_groups := NewGroups().AddFromString(whitelist)
		l1 := configured_groups.Len()
		combined_groups := configured_groups.Add(groups)
		l2 := combined_groups.Len()
		//fmt.Printf("configured: %d, combined: %d", l1, l2)
		if l1 == l2 {
			return nil
		}
		cm.Data["whitelist.txt"] = combined_groups.AsString()
	}
	return r.Update(context.TODO(), cm)

}

// ensureLdapGroup ensures Group presence
func (r *PaasReconciler) FinalizeLdapGroups(
	paas *mydomainv1alpha1.Paas,
	cleanedLdapQueries []string,
) error {
	// See if group already exists and create if it doesn't
	cm := &corev1.ConfigMap{}
	wlConfigMap := CaasWhiteList()
	err := r.Get(context.TODO(), wlConfigMap, cm)
	if err != nil && errors.IsNotFound(err) {
		// ConfigMap does not exist, so nothing to clean
		return nil
	} else if err != nil {
		// Error that isn't due to the group not existing
		return err
	} else if whitelist, exists := cm.Data["whitelist.txt"]; !exists {
		// No whitelist.txt exists in the configmap, so nothing to clean
		return nil
	} else {
		var isChanged bool
		groups := NewGroups().AddFromString(whitelist)
		for _, query := range cleanedLdapQueries {
			if groups.DeleteByValue(query) {
				isChanged = true
			}
		}
		//fmt.Printf("configured: %d, combined: %d", l1, l2)
		if !isChanged {
			return nil
		}
		cm.Data["whitelist.txt"] = groups.AsString()
	}
	return r.Update(context.TODO(), cm)

}
