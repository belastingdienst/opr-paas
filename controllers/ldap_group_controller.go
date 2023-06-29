package controllers

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
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
	ctx context.Context,
	whiteListConfigMap types.NamespacedName,
	groups *Groups,
) error {
	// Create the ConfigMap
	wlConfigMap := CaasWhiteList()
	return r.Create(ctx, &corev1.ConfigMap{
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

func (gs *Groups) DeleteByKey(key string) bool {
	if _, exists := gs.by_key[key]; exists {
		delete(gs.by_key, key)
		return true
	}
	return false
}

func (gs *Groups) DeleteByQuery(query string) bool {
	for key, value := range gs.by_key {
		if value == query {
			delete(gs.by_key, key)
			return true
		}
	}
	return false
}

func (gs Groups) Len() int {
	return len(gs.by_key)
}

func (gs *Groups) Add(other *Groups) bool {
	var changed bool
	for key, value := range other.by_key {
		if newVal, exists := gs.by_key[key]; !exists {
			changed = true
		} else if newVal != value {
			changed = true
		}
		gs.by_key[key] = value
	}
	return changed
}

func QueryToKey(query string) string {
	//CN=gkey,OU=org_unit,DC=example,DC=org
	if cn := strings.Split(query, ",")[0]; !strings.ContainsAny(cn, "=") {
		return ""
	} else {
		return strings.SplitN(cn, "=", 2)[1]
	}

}
func (gs *Groups) AddFromStrings(l []string) {
	for _, query := range l {
		if key := QueryToKey(query); key == "" {
			continue
		} else {
			gs.by_key[key] = query
		}
	}
}

func (gs *Groups) AddFromString(s string) {
	gs.AddFromStrings(strings.Split(s, "\n"))
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
	ctx context.Context,
	paas *v1alpha1.Paas,
) error {
	logger := getLogger(ctx, paas, "LdapGroup", "")
	// See if group already exists and create if it doesn't
	cm := &corev1.ConfigMap{}
	wlConfigMap := CaasWhiteList()
	err := r.Get(ctx, wlConfigMap, cm)
	groups := NewGroups()
	groups.AddFromStrings(paas.Spec.Groups.LdapQueries())
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating whitelist configmap")
		// Create the ConfigMap
		return r.ensureLdapGroupsConfigMap(ctx, wlConfigMap, groups)
	} else if err != nil {
		logger.Error(err, "Could not retrieve whitelist configmap")
		// Error that isn't due to the group not existing
		return err
	} else if whitelist, exists := cm.Data["whitelist.txt"]; !exists {
		logger.Info("Adding whitelist.txt to whitelist configmap")
		cm.Data["whitelist.txt"] = groups.AsString()
	} else {
		logger.Info(fmt.Sprintf("Reading group queries from whitelist %v", cm))
		whitelist_groups := NewGroups()
		whitelist_groups.AddFromString(whitelist)
		logger.Info(fmt.Sprintf("Adding extra groups to whitelist: %v", groups))
		if changed := whitelist_groups.Add(groups); changed {
			//fmt.Printf("configured: %d, combined: %d", l1, l2)
			logger.Info("No new info in whitelist")
			return nil
		}
		logger.Info("Adding to whitelist configmap")
		cm.Data["whitelist.txt"] = whitelist_groups.AsString()
	}
	logger.Info(fmt.Sprintf("Updating whitelist configmap: %v", cm))
	return r.Update(ctx, cm)
}

// ensureLdapGroup ensures Group presence
func (r *PaasReconciler) FinalizeLdapGroups(
	ctx context.Context,
	paas *v1alpha1.Paas,
	cleanedLdapQueries []string,
) error {
	logger := getLogger(ctx, paas, "LdapGroup", "")
	// See if group already exists and create if it doesn't
	cm := &corev1.ConfigMap{}
	wlConfigMap := CaasWhiteList()
	err := r.Get(ctx, wlConfigMap, cm)
	if err != nil && errors.IsNotFound(err) {
		logger.Error(err, "whitelist configmap does not exist")
		// ConfigMap does not exist, so nothing to clean
		return nil
	} else if err != nil {
		logger.Error(err, "error retrieving whitelist configmap")
		// Error that isn't due to the group not existing
		return err
	} else if whitelist, exists := cm.Data["whitelist.txt"]; !exists {
		// No whitelist.txt exists in the configmap, so nothing to clean
		logger.Error(fmt.Errorf("no whitelist"), "whitelist.txt does not exists in whitelist configmap")
		return nil
	} else {
		var isChanged bool
		groups := NewGroups()
		groups.AddFromString(whitelist)
		for _, query := range cleanedLdapQueries {
			if key := QueryToKey(query); key == "" {
				logger.Error(fmt.Errorf("invalid query"), "Could not get key from", query)
			} else if groups.DeleteByKey(key) {
				logger.Info(fmt.Sprintf("LdapGroup %s removed", key))
				isChanged = true
			}
		}
		//fmt.Printf("configured: %d, combined: %d", l1, l2)
		if !isChanged {
			return nil
		}
		cm.Data["whitelist.txt"] = groups.AsString()
	}
	return r.Update(ctx, cm)

}
