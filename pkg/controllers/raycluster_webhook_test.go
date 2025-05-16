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
	"testing"

	. "github.com/onsi/gomega"
	"github.com/project-codeflare/codeflare-common/support"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/project-codeflare/codeflare-operator/pkg/config"
)

var (
	namespace      = "test-namespace"
	rayClusterName = "test-raycluster"

	rcWebhook = &rayClusterWebhook{
		Config:          &config.KubeRayConfiguration{},
		OperatorVersion: "0.0.0",
	}
)

func TestRayClusterWebhookDefault(t *testing.T) {
	test := support.NewTest(t)

	validRayCluster := &rayv1.RayCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rayClusterName,
			Namespace: namespace,
		},
		Spec: rayv1.RayClusterSpec{
			HeadGroupSpec: rayv1.HeadGroupSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "ray-head"}},
					},
				},
				RayStartParams: map[string]string{},
			},
			WorkerGroupSpecs: []rayv1.WorkerGroupSpec{
				{
					GroupName: "worker-group-1",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "worker-container-1",
								},
							},
						},
					},
					RayStartParams: map[string]string{},
				},
			},
		},
	}

	// Create the RayClusters
	if _, err := test.Client().Ray().RayV1().RayClusters(namespace).Create(test.Ctx(), validRayCluster, metav1.CreateOptions{}); err != nil {
		test.T().Fatalf("Failed to create RayCluster: %v", err)
	}

	// Call to default function is made
	err := rcWebhook.Default(test.Ctx(), runtime.Object(validRayCluster))
	t.Run("Expected no errors on call to Default function", func(t *testing.T) {
		test.Expect(err).ShouldNot(HaveOccurred(), "Expected no errors on call to Default function")
	})

	t.Run("Expected required mTLS volumes for the head group", func(t *testing.T) {
		test.Expect(validRayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes).
			To(
				And(
					HaveLen(2),
					ContainElement(WithTransform(support.ResourceName, Equal("ca-vol"))),
					ContainElement(WithTransform(support.ResourceName, Equal("server-cert"))),
				),
				"Expected the mTLS CA and server-cert volumes to be present in the head group",
			)
	})

	t.Run("Expected required mTLS environment variables for the head group", func(t *testing.T) {
		test.Expect(validRayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].Env).
			To(
				And(
					HaveLen(5),
					ContainElement(WithTransform(support.ResourceName, Equal("MY_POD_IP"))),
					ContainElement(WithTransform(support.ResourceName, Equal("RAY_USE_TLS"))),
					ContainElement(WithTransform(support.ResourceName, Equal("RAY_TLS_SERVER_CERT"))),
					ContainElement(WithTransform(support.ResourceName, Equal("RAY_TLS_SERVER_KEY"))),
					ContainElement(WithTransform(support.ResourceName, Equal("RAY_TLS_CA_CERT"))),
				),
				"Expected the required mTLS environment variables to be present in the head group container",
			)
	})

	t.Run("Expected required create-cert init container for the head group for mTLS", func(t *testing.T) {
		test.Expect(validRayCluster.Spec.HeadGroupSpec.Template.Spec.InitContainers).
			To(
				And(
					HaveLen(1),
					ContainElement(WithTransform(support.ResourceName, Equal(initContainerName))),
				),
				"Expected the create-cert init container to be present in the head group for mTLS",
			)
	})

	t.Run("Expected required mTLS volume mounts for the head group container", func(t *testing.T) {
		test.Expect(validRayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].VolumeMounts).
			To(
				And(
					HaveLen(2),
					ContainElement(WithTransform(support.ResourceName, Equal("ca-vol"))),
					ContainElement(WithTransform(support.ResourceName, Equal("server-cert"))),
				),
				"Expected the mTLS CA and server-cert volume mounts to be present in the head group container",
			)
	})

	t.Run("Expected required environment variables for each worker group", func(t *testing.T) {
		for _, workerGroup := range validRayCluster.Spec.WorkerGroupSpecs {
			test.Expect(workerGroup.Template.Spec.Containers[0].Env).
				To(
					And(
						HaveLen(5),
						ContainElement(WithTransform(support.ResourceName, Equal("MY_POD_IP"))),
						ContainElement(WithTransform(support.ResourceName, Equal("RAY_USE_TLS"))),
						ContainElement(WithTransform(support.ResourceName, Equal("RAY_TLS_SERVER_CERT"))),
						ContainElement(WithTransform(support.ResourceName, Equal("RAY_TLS_SERVER_KEY"))),
						ContainElement(WithTransform(support.ResourceName, Equal("RAY_TLS_CA_CERT"))),
					),
					"Expected the required environment variables to be present in each worker group",
				)
		}
	})

	t.Run("Expected required CA Volumes for each worker group", func(t *testing.T) {
		for _, workerGroup := range validRayCluster.Spec.WorkerGroupSpecs {
			test.Expect(workerGroup.Template.Spec.Volumes).
				To(
					And(
						HaveLen(2),
						ContainElement(WithTransform(support.ResourceName, Equal("ca-vol"))),
						ContainElement(WithTransform(support.ResourceName, Equal("server-cert"))),
					),
					"Expected the required CA volumes to be present in each worker group",
				)
		}
	})

	t.Run("Expected required certificate volume mounts for each worker group", func(t *testing.T) {
		for _, workerSpec := range validRayCluster.Spec.WorkerGroupSpecs {
			test.Expect(workerSpec.Template.Spec.Containers[0].VolumeMounts).
				To(
					And(
						HaveLen(2),
						ContainElement(WithTransform(support.ResourceName, Equal("ca-vol"))),
						ContainElement(WithTransform(support.ResourceName, Equal("server-cert"))),
					),
					"Expected the required certificate volume mounts to be present in each worker group",
				)
		}
	})

	t.Run("Expected required init container for each worker group", func(t *testing.T) {
		for _, workerSpec := range validRayCluster.Spec.WorkerGroupSpecs {
			test.Expect(workerSpec.Template.Spec.InitContainers).
				To(
					And(
						HaveLen(1),
						ContainElement(WithTransform(support.ResourceName, Equal(initContainerName))),
					),
					"Expected the required init container to be present in each worker group",
				)
		}
	})

}

func TestValidateCreate(t *testing.T) {
	test := support.NewTest(t)
	minimalRayCluster := &rayv1.RayCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rayClusterName,
			Namespace: namespace,
		},
		Spec: rayv1.RayClusterSpec{
			HeadGroupSpec: rayv1.HeadGroupSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "head"},
						},
					},
				},
				RayStartParams: map[string]string{},
			},
			WorkerGroupSpecs: []rayv1.WorkerGroupSpec{
				{
					GroupName: "worker-group-1",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "worker-container-1",
								},
							},
						},
					},
					RayStartParams: map[string]string{},
				},
			},
		},
	}

	validRayClusterForMTLS := &rayv1.RayCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rayClusterName,
			Namespace: namespace,
			Annotations: map[string]string{
				versionAnnotation: "0.0.0",
			},
		},
		Spec: rayv1.RayClusterSpec{
			HeadGroupSpec: rayv1.HeadGroupSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "ray-head",
								Image: "rayproject/ray:latest",
							},
						},
					},
				},
				RayStartParams: map[string]string{},
			},
		},
	}

	// Create the RayClusters
	if _, err := test.Client().Ray().RayV1().RayClusters(namespace).Create(test.Ctx(), minimalRayCluster, metav1.CreateOptions{}); err != nil {
		test.T().Fatalf("Failed to create RayCluster: %v", err)
	}

	t.Run("Expect updated values in minimal RayCluster after Defaulting", func(t *testing.T) {
		dCopy := minimalRayCluster.DeepCopy()
		err := rcWebhook.Default(test.Ctx(), runtime.Object(dCopy))
		test.Expect(err).ShouldNot(HaveOccurred(), "Expected no errors on call to Default function")
		test.Expect(dCopy.GetAnnotations()[versionAnnotation]).ShouldNot(BeNil(), "Expected version annotation to be set")
	})

	warnings, err := rcWebhook.ValidateCreate(test.Ctx(), runtime.Object(minimalRayCluster))
	t.Run("Expected no warnings or errors on call to ValidateCreate function for minimal cluster", func(t *testing.T) {
		test.Expect(warnings).Should(BeNil(), "Expected no warnings on call to ValidateCreate function")
		test.Expect(err).ShouldNot(HaveOccurred(), "Expected no errors on call to ValidateCreate function")
	})

	t.Run("Negative: Expected errors on call to ValidateCreate function due to EnableIngress set to True", func(t *testing.T) {
		invalidRayCluster := validRayClusterForMTLS.DeepCopy()
		rcWebhook.Default(test.Ctx(), runtime.Object(invalidRayCluster))
		invalidRayCluster.Spec.HeadGroupSpec.EnableIngress = support.Ptr(true)
		_, err := rcWebhook.ValidateCreate(test.Ctx(), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateCreate function due to EnableIngress set to True")
	})

}

func TestValidateUpdate(t *testing.T) {
	test := support.NewTest(t)
	// rayClientRoute and svcDomain are no longer used in the refactored test setup for baseRayClusterWithMTLS
	// rayClientRoute := "rayclient-" + rayClusterName + "-" + namespace + "." + rcWebhook.Config.IngressDomain
	// svcDomain := rayClusterName + "-head-svc." + namespace + ".svc"

	baseRayClusterWithMTLS := &rayv1.RayCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rayClusterName,
			Namespace: namespace,
			Annotations: map[string]string{
				versionAnnotation: "0.0.0",
			},
		},
		Spec: rayv1.RayClusterSpec{
			HeadGroupSpec: rayv1.HeadGroupSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:         "ray-head",
								Image:        "rayproject/ray:latest",
								Env:          envVarList(),
								VolumeMounts: certVolumeMounts(),
							},
						},
						InitContainers: []corev1.Container{
							rayHeadInitContainer(&rayv1.RayCluster{ObjectMeta: metav1.ObjectMeta{Name: rayClusterName, Namespace: namespace}}, rcWebhook.Config),
						},
						Volumes: caVolumes(&rayv1.RayCluster{ObjectMeta: metav1.ObjectMeta{Name: rayClusterName, Namespace: namespace}}),
					},
				},
				RayStartParams: map[string]string{},
			},
			WorkerGroupSpecs: []rayv1.WorkerGroupSpec{
				{
					GroupName: "worker-group-1",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:         "worker-container-1",
									Env:          envVarList(),
									VolumeMounts: certVolumeMounts(),
								},
							},
							InitContainers: []corev1.Container{
								rayWorkerInitContainer(rcWebhook.Config),
							},
							Volumes: caVolumes(&rayv1.RayCluster{ObjectMeta: metav1.ObjectMeta{Name: rayClusterName, Namespace: namespace}}),
						},
					},
					RayStartParams: map[string]string{},
				},
			},
		},
	}

	validRayClusterOldNames := baseRayClusterWithMTLS.DeepCopy()
	validRayClusterOldNames.Annotations = nil

	if _, err := test.Client().Ray().RayV1().RayClusters(namespace).Create(test.Ctx(), baseRayClusterWithMTLS, metav1.CreateOptions{}); err != nil {
		test.T().Fatalf("Failed to create RayCluster: %v", err)
	}

	t.Run("Expected no warnings or errors on call to ValidateUpdate function with version annotation set", func(t *testing.T) {
		test.Expect(runtime.Object(baseRayClusterWithMTLS).(*rayv1.RayCluster).Annotations).ShouldNot(BeNil(), "Expected version annotation to be set")
		warnings, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(baseRayClusterWithMTLS), runtime.Object(baseRayClusterWithMTLS))
		test.Expect(warnings).Should(BeNil(), "Expected no warnings on call to ValidateUpdate function")
		test.Expect(err).ShouldNot(HaveOccurred(), "Expected no errors on call to ValidateUpdate function")
	})

	t.Run("Expected no warnings or errors on call to ValidateUpdate function with version annotation unset", func(t *testing.T) {
		warnings, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(validRayClusterOldNames), runtime.Object(validRayClusterOldNames))
		test.Expect(warnings).Should(BeNil(), "Expected no warnings on call to ValidateUpdate function")
		test.Expect(err).ShouldNot(HaveOccurred(), "Expected no errors on call to ValidateUpdate function")
	})

	t.Run("Negative: Expected errors on call to ValidateUpdate function due to EnableIngress set to True", func(t *testing.T) {
		invalidRayCluster := baseRayClusterWithMTLS.DeepCopy()
		invalidRayCluster.Spec.HeadGroupSpec.EnableIngress = support.Ptr(true)
		_, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(baseRayClusterWithMTLS), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateUpdate function due to EnableIngress set to True")
	})

	t.Run("Negative: Expected errors on call to ValidateUpdate function due to manipulated Init Container in the head group", func(t *testing.T) {
		invalidRayCluster := baseRayClusterWithMTLS.DeepCopy()
		for i, headInitContainer := range invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.InitContainers {
			if headInitContainer.Name == "create-cert" {
				invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.InitContainers[i].Command = []string{"manipulated command"}
				break
			}
		}
		_, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(baseRayClusterWithMTLS), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateUpdate function due to manipulated Init Container in the head group")
	})

	t.Run("Negative: Expected errors on call to ValidateUpdate function due to manipulated Init Container in the worker group", func(t *testing.T) {
		invalidRayCluster := baseRayClusterWithMTLS.DeepCopy()
		for _, workerGroup := range invalidRayCluster.Spec.WorkerGroupSpecs {
			for i, workerInitContainer := range workerGroup.Template.Spec.InitContainers {
				if workerInitContainer.Name == "create-cert" {
					workerGroup.Template.Spec.InitContainers[i].Command = []string{"manipulated command"}
					break
				}
			}
		}
		_, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(baseRayClusterWithMTLS), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateUpdate function due to manipulated Init Container in the worker group")
	})

	t.Run("Negative: Expected errors on call to ValidateUpdate function due to manipulated Volume in the head group", func(t *testing.T) {
		invalidRayCluster := baseRayClusterWithMTLS.DeepCopy()
		for i, headVolume := range invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes {
			if headVolume.Name == "ca-vol" {
				invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes[i].Secret.SecretName = "invalid-secret-name"
				break
			}
		}
		_, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(baseRayClusterWithMTLS), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateUpdate function due to manipulated Volume in the head group")
	})

	t.Run("Negative: Expected errors on call to ValidateUpdate function due to manipulated Volume in the worker group", func(t *testing.T) {
		invalidRayCluster := baseRayClusterWithMTLS.DeepCopy()
		for _, workerGroup := range invalidRayCluster.Spec.WorkerGroupSpecs {
			for i, workerVolume := range workerGroup.Template.Spec.Volumes {
				if workerVolume.Name == "ca-vol" {
					workerGroup.Template.Spec.Volumes[i].Secret.SecretName = "invalid-secret-name"
					break
				}
			}
		}
		_, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(baseRayClusterWithMTLS), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateUpdate function due to manipulated Volume in the worker group")
	})

	t.Run("Negative: Expected errors on call to ValidateUpdate function due to manipulated env vars in the head group", func(t *testing.T) {
		invalidRayCluster := baseRayClusterWithMTLS.DeepCopy()
		for i, headEnvVar := range invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].Env {
			if headEnvVar.Name == "RAY_USE_TLS" {
				invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].Env[i].Value = "invalid-value"
				break
			}
		}
		_, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(baseRayClusterWithMTLS), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateUpdate function due to manipulated env vars in the head group")
	})

	t.Run("Negative: Expected errors on call to ValidateUpdate function due to manipulated env vars in the worker group", func(t *testing.T) {
		invalidRayCluster := baseRayClusterWithMTLS.DeepCopy()
		for _, workerGroup := range invalidRayCluster.Spec.WorkerGroupSpecs {
			for i, workerEnvVar := range workerGroup.Template.Spec.Containers[0].Env {
				if workerEnvVar.Name == "RAY_USE_TLS" {
					workerGroup.Template.Spec.Containers[0].Env[i].Value = "invalid-value"
					break
				}
			}
		}
		_, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(baseRayClusterWithMTLS), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateUpdate function due to manipulated env vars in the worker group")
	})
}
