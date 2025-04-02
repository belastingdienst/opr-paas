package v1alpha2

import (
	"testing"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/belastingdienst/opr-paas/internal/quota"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var exV1Alpha1 = &v1alpha1.Paas{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "foo",
		Namespace: "bar",
	},
	Spec: v1alpha1.PaasSpec{
		Requestor: "some-requestor",
		Quota: quota.Quota{
			corev1.ResourceLimitsCPU:      resource.MustParse("2"),
			corev1.ResourceLimitsMemory:   resource.MustParse("2Gi"),
			corev1.ResourceRequestsCPU:    resource.MustParse("500m"),
			corev1.ResourceRequestsMemory: resource.MustParse("256Mi"),
		},
		Capabilities: v1alpha1.PaasCapabilities{
			"argocd": {
				Enabled:     true,
				GitURL:      "ssh://git@example.com/some-repo.git",
				GitRevision: "main",
				GitPath:     ".",
				CustomFields: map[string]string{
					"field1": "value",
				},
				Quota: quota.Quota{
					corev1.ResourceRequestsCPU: resource.MustParse("250m"),
				},
				SSHSecrets: map[string]string{
					"secret1": "ZW5jcnlwdGVkIHZhbHVlCg==",
				},
				ExtraPermissions: true,
			},
		},
		Groups: v1alpha1.PaasGroups{
			"some-group": {
				Query: "some query",
				Users: []string{"user1", "user2"},
				Roles: []string{"role1", "role2"},
			},
		},
		Namespaces: []string{
			"namespace1",
			"namespace2",
		},
		ManagedByPaas: "some other paas",
	},
	Status: v1alpha1.PaasStatus{
		Conditions: []metav1.Condition{
			{
				Type:   TypeReadyPaas,
				Status: metav1.ConditionTrue,
			},
			{
				Type:   TypeHasErrorsPaas,
				Status: metav1.ConditionFalse,
			},
		},
	},
}
var exV1Alpha2 = &Paas{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "foo",
		Namespace: "bar",
	},
	Spec: PaasSpec{
		Requestor: "some-requestor",
		Quota: quota.Quota{
			corev1.ResourceLimitsCPU:      resource.MustParse("2"),
			corev1.ResourceLimitsMemory:   resource.MustParse("2Gi"),
			corev1.ResourceRequestsCPU:    resource.MustParse("500m"),
			corev1.ResourceRequestsMemory: resource.MustParse("256Mi"),
		},
		Capabilities: PaasCapabilities{
			"argocd": {
				CustomFields: map[string]string{
					"field1":       "value",
					"git_url":      "ssh://git@example.com/some-repo.git",
					"git_revision": "main",
					"git_path":     ".",
				},
				Quota: quota.Quota{
					corev1.ResourceRequestsCPU: resource.MustParse("250m"),
				},
				Secrets: map[string]string{
					"secret1": "ZW5jcnlwdGVkIHZhbHVlCg==",
				},
				ExtraPermissions: true,
			},
		},
		Groups: PaasGroups{
			"some-group": {
				"some query",
				[]string{"user1", "user2"},
				[]string{"role1", "role2"},
			},
		},
		Namespaces: PaasNamespaces{
			"namespace1": {},
			"namespace2": {},
		},
		ManagedByPaas: "some other paas",
	},
	Status: PaasStatus{
		Conditions: []metav1.Condition{
			{
				Type:   TypeReadyPaas,
				Status: metav1.ConditionTrue,
			},
			{
				Type:   TypeHasErrorsPaas,
				Status: metav1.ConditionFalse,
			},
		},
	},
}

func TestConvertTo(t *testing.T) {
	src := exV1Alpha2.DeepCopy()
	dst := &v1alpha1.Paas{}

	err := src.ConvertTo(dst)

	assert.NoError(t, err)
	assert.Equal(t, exV1Alpha1, dst)
}

func TestConvertFrom(t *testing.T) {
	src := exV1Alpha1.DeepCopy()
	dst := &Paas{}

	err := dst.ConvertFrom(src)

	assert.NoError(t, err)
	assert.Equal(t, exV1Alpha2, dst)
}
