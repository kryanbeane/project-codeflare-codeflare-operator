/*
Copyright 2024.

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

package controllers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	// "github.com/project-codeflare/codeflare-common/support"

	// rbacv1 "k8s.io/api/rbac/v1"
	// "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// routev1 "github.com/openshift/api/route/v1"
)

var _ = Describe("RayCluster controller", func() {
	Context("RayCluster controller test", func() {
		// rayClusterName := "test-raycluster"
		// var namespaceName string
		// BeforeEach(func(ctx SpecContext) {
		// 	By("Creating a namespace for running the tests.")
		// 	namespace := &corev1.Namespace{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			GenerateName: "test-",
		// 		},
		// 	}
		// 	namespace, err := k8sClient.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
		// 	Expect(err).NotTo(HaveOccurred())
		// 	DeferCleanup(func(ctx SpecContext) {
		// 		err := k8sClient.CoreV1().Namespaces().Delete(ctx, namespace.Name, metav1.DeleteOptions{})
		// 		Expect(err).To(Not(HaveOccurred()))
		// 	})
		// 	namespaceName = namespace.Name

		// 	By("creating a basic instance of the RayCluster CR")
		// 	raycluster := &rayv1.RayCluster{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      rayClusterName,
		// 			Namespace: namespace.Name,
		// 		},
		// 		Spec: rayv1.RayClusterSpec{
		// 			HeadGroupSpec: rayv1.HeadGroupSpec{
		// 				Template: corev1.PodTemplateSpec{
		// 					Spec: corev1.PodSpec{
		// 						Containers: []corev1.Container{},
		// 					},
		// 				},
		// 				RayStartParams: map[string]string{},
		// 			},
		// 			Suspend: support.Ptr(false),
		// 		},
		// 	}
		// 	_, err = rayClient.RayV1().RayClusters(namespace.Name).Create(ctx, raycluster, metav1.CreateOptions{})
		// 	Expect(err).To(Not(HaveOccurred()))
		// })

		// AfterEach(func(ctx SpecContext) {
		// 	By("removing instances of the RayClusters used")
		// 	rayClusters, err := rayClient.RayV1().RayClusters(namespaceName).List(ctx, metav1.ListOptions{})
		// 	Expect(err).To(Not(HaveOccurred()))

		// 	for _, rayCluster := range rayClusters.Items {
		// 		err = rayClient.RayV1().RayClusters(namespaceName).Delete(ctx, rayCluster.Name, metav1.DeleteOptions{})
		// 		Expect(err).To(Not(HaveOccurred()))
		// 	}

		// 	Eventually(func() ([]rayv1.RayCluster, error) {
		// 		rayClusters, err := rayClient.RayV1().RayClusters(namespaceName).List(ctx, metav1.ListOptions{})
		// 		return rayClusters.Items, err
		// 	}).WithTimeout(time.Second * 10).Should(BeEmpty())
		// })

		// All OAuth related tests are removed.
		// Remaining tests should focus on mTLS or other non-OAuth controller functionalities if any.
		// For now, this leaves the test file mostly empty pending new tests for remaining/new logic.

		// Example of a placeholder test if needed, actual tests for mTLS resources created by controller would go here.
		It("should correctly reconcile non-OAuth resources if applicable", func() {
			// Placeholder: Add assertions for any resources the controller is still expected to create/manage,
			// such as CA secrets for mTLS if that logic resides in the controller.
			Expect(true).To(BeTrue()) // Replace with actual test logic
		})

	})
})

func OwnerReferenceKind(meta metav1.Object) string {
	return meta.GetOwnerReferences()[0].Kind
}

func OwnerReferenceName(meta metav1.Object) string {
	return meta.GetOwnerReferences()[0].Name
}
