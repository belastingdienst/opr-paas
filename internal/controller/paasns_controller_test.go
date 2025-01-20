/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	api "github.com/belastingdienst/opr-paas/api/v1alpha1"
)

var _ = Describe("PaasNS Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()

		const name = "my-paas-ns"
		paasns := &api.PaasNS{
			ObjectMeta: meta.ObjectMeta{
				Name:      name,
				Namespace: "default",
			},
			Spec: api.PaasNSSpec{Paas: "foo"},
		}

		BeforeEach(func() {
			By("creating PaasNS " + name)
			Expect(k8sClient.Create(ctx, paasns)).
				To(Succeed())
		})

		AfterEach(func() {
			By("deleting PaasNS " + name)
			Expect(k8sClient.Delete(ctx, paasns)).
				To(Succeed())
		})

		It("should not return an error", func() {
			nsname := types.NamespacedName{
				Name:      name,
				Namespace: "default",
			}
			reconciler := &PaasNSReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := reconciler.Reconcile(
				ctx,
				reconcile.Request{NamespacedName: nsname},
			)

			Expect(err).NotTo(HaveOccurred())
		})
	})
})
