/*
Copyright 2023.

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
	"context"
	"strconv"

	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/project-codeflare/codeflare-operator/pkg/config"
)

const (
	initContainerName = "create-cert"
	versionAnnotation = "ray.openshift.ai/version"
)

// log is for logging in this package.
var rayclusterlog = logf.Log.WithName("raycluster-resource")

func SetupRayClusterWebhookWithManager(mgr ctrl.Manager, cfg *config.KubeRayConfiguration, operatorVersion string) error {
	rayClusterWebhookInstance := &rayClusterWebhook{
		Config:          cfg,
		OperatorVersion: operatorVersion,
	}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&rayv1.RayCluster{}).
		WithDefaulter(rayClusterWebhookInstance).
		WithValidator(rayClusterWebhookInstance).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-ray-io-v1-raycluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=ray.io,resources=rayclusters,verbs=create;update,versions=v1,name=mraycluster.ray.openshift.ai,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-ray-io-v1-raycluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=ray.io,resources=rayclusters,verbs=create;update,versions=v1,name=vraycluster.ray.openshift.ai,admissionReviewVersions=v1

type rayClusterWebhook struct {
	Config          *config.KubeRayConfiguration
	OperatorVersion string
}

var _ webhook.CustomDefaulter = &rayClusterWebhook{}
var _ webhook.CustomValidator = &rayClusterWebhook{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (w *rayClusterWebhook) Default(ctx context.Context, obj runtime.Object) error {
	rayCluster := obj.(*rayv1.RayCluster)
	rayclusterlog.Info("Defaulting RayCluster", "name", rayCluster.Name)

	// Set OperatorVersion Annotation
	annotations := rayCluster.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[versionAnnotation] = w.OperatorVersion
	rayCluster.SetAnnotations(annotations)

	// Defaulting for mTLS if enabled
	if ptr.Deref(w.Config.MTLSEnabled, true) {
		rayclusterlog.V(2).Info("Adding create-cert Init Containers and TLS env vars for mTLS")
		// HeadGroupSpec
		if len(rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers) > 0 {
			for _, envVar := range envVarList() {
				rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].Env = upsert(rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].Env, envVar, withEnvVarName(envVar.Name))
			}
			rayCluster.Spec.HeadGroupSpec.Template.Spec.InitContainers = upsert(rayCluster.Spec.HeadGroupSpec.Template.Spec.InitContainers, rayHeadInitContainer(rayCluster, w.Config), withContainerName(initContainerName))
			for _, caVol := range caVolumes(rayCluster) {
				rayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes = upsert(rayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes, caVol, withVolumeName(caVol.Name))
			}
			for _, mount := range certVolumeMounts() {
				rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].VolumeMounts = upsert(rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].VolumeMounts, mount, byVolumeMountName)
			}
		} else {
			rayclusterlog.Info("Warning: HeadGroupSpec.Template.Spec.Containers is empty. Cannot apply mTLS defaults.")
		}

		// WorkerGroupSpec
		for i := range rayCluster.Spec.WorkerGroupSpecs {
			workerSpec := &rayCluster.Spec.WorkerGroupSpecs[i]
			if len(workerSpec.Template.Spec.Containers) > 0 {
				for _, envVar := range envVarList() {
					workerSpec.Template.Spec.Containers[0].Env = upsert(workerSpec.Template.Spec.Containers[0].Env, envVar, withEnvVarName(envVar.Name))
				}
				for _, caVol := range caVolumes(rayCluster) {
					workerSpec.Template.Spec.Volumes = upsert(workerSpec.Template.Spec.Volumes, caVol, withVolumeName(caVol.Name))
				}
				for _, mount := range certVolumeMounts() {
					workerSpec.Template.Spec.Containers[0].VolumeMounts = upsert(workerSpec.Template.Spec.Containers[0].VolumeMounts, mount, byVolumeMountName)
				}
				workerSpec.Template.Spec.InitContainers = upsert(workerSpec.Template.Spec.InitContainers, rayWorkerInitContainer(w.Config), withContainerName(initContainerName))
			} else {
				rayclusterlog.Info("Warning: WorkerGroupSpecs.Template.Spec.Containers is empty. Cannot apply mTLS defaults.", "workerGroup", workerSpec.GroupName)
			}
		}
	}

	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (w *rayClusterWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	rayCluster := obj.(*rayv1.RayCluster)
	rayclusterlog.Info("ValidateCreate RayCluster", "name", rayCluster.Name)
	var warnings admission.Warnings
	var allErrors field.ErrorList

	allErrors = append(allErrors, validateIngress(rayCluster)...)

	return warnings, allErrors.ToAggregate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (w *rayClusterWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldRayCluster := oldObj.(*rayv1.RayCluster)
	newRayCluster := newObj.(*rayv1.RayCluster)
	rayclusterlog.Info("ValidateUpdate RayCluster", "name", newRayCluster.Name)
	var warnings admission.Warnings
	var allErrors field.ErrorList

	allErrors = append(allErrors, validateIngress(newRayCluster)...)

	if ptr.Deref(w.Config.MTLSEnabled, true) {
		if !equality.Semantic.DeepEqual(oldRayCluster.Spec.HeadGroupSpec.Template.Spec.InitContainers, newRayCluster.Spec.HeadGroupSpec.Template.Spec.InitContainers) {
			allErrors = append(allErrors, validateHeadInitContainers(newRayCluster, w.Config)...)
		}
		if !equality.Semantic.DeepEqual(oldRayCluster.Spec.WorkerGroupSpecs, newRayCluster.Spec.WorkerGroupSpecs) {
			allErrors = append(allErrors, validateWorkerInitContainers(newRayCluster, w.Config)...)
		}
		if !equality.Semantic.DeepEqual(oldRayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].Env, newRayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].Env) {
			allErrors = append(allErrors, validateHeadEnvVars(newRayCluster)...)
		}
		if !equality.Semantic.DeepEqual(oldRayCluster.Spec.WorkerGroupSpecs, newRayCluster.Spec.WorkerGroupSpecs) {
			allErrors = append(allErrors, validateWorkerEnvVars(newRayCluster)...)
		}
	}

	return warnings, allErrors.ToAggregate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (w *rayClusterWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}

func validateIngress(rayCluster *rayv1.RayCluster) field.ErrorList {
	var allErrors field.ErrorList

	if ptr.Deref(rayCluster.Spec.HeadGroupSpec.EnableIngress, false) {
		allErrors = append(allErrors, field.Invalid(
			field.NewPath("spec", "headGroupSpec", "enableIngress"),
			rayCluster.Spec.HeadGroupSpec.EnableIngress,
			"RayCluster with enableIngress set to true is not allowed"))
	}

	return allErrors
}

func envVarList() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: "MY_POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
		{
			Name:  "RAY_USE_TLS",
			Value: "1",
		},
		{
			Name:  "RAY_TLS_SERVER_CERT",
			Value: "/home/ray/workspace/tls/server.crt",
		},
		{
			Name:  "RAY_TLS_SERVER_KEY",
			Value: "/home/ray/workspace/tls/server.key",
		},
		{
			Name:  "RAY_TLS_CA_CERT",
			Value: "/home/ray/workspace/tls/ca.crt",
		},
	}
}

func caVolumes(rayCluster *rayv1.RayCluster) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "ca-vol",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: caSecretNameFromCluster(rayCluster),
				},
			},
		},
		{
			Name: "server-cert",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}

func rayHeadInitContainer(rayCluster *rayv1.RayCluster, config *config.KubeRayConfiguration) corev1.Container {
	rayClientRoute := rayClientNameFromCluster(rayCluster) + "-" + rayCluster.Namespace + "." + config.IngressDomain
	// Service name for basic interactive
	svcDomain := serviceNameFromCluster(rayCluster) + "." + rayCluster.Namespace + ".svc"

	initContainerHead := corev1.Container{
		Name:  "create-cert",
		Image: CertGeneratorImage,
		Command: []string{
			"sh",
			"-c",
			`cd /home/ray/workspace/tls && openssl req -nodes -newkey rsa:2048 -keyout server.key -out server.csr -subj '/CN=ray-head' && printf "authorityKeyIdentifier=keyid,issuer\nbasicConstraints=CA:FALSE\nsubjectAltName = @alt_names\n[alt_names]\nDNS.1 = 127.0.0.1\nDNS.2 = localhost\nDNS.3 = ${FQ_RAY_IP}\nDNS.4 = $(awk 'END{print $1}' /etc/hosts)\nDNS.5 = ` + rayClientRoute + `\nDNS.6 = ` + svcDomain + `">./domain.ext && cp /home/ray/workspace/ca/* . && openssl x509 -req -CA ca.crt -CAkey ca.key -in server.csr -out server.crt -days 365 -CAcreateserial -extfile domain.ext`,
		},
		VolumeMounts: certVolumeMounts(),
	}
	return initContainerHead
}

func rayWorkerInitContainer(config *config.KubeRayConfiguration) corev1.Container {
	initContainerWorker := corev1.Container{
		Name:  "create-cert",
		Image: CertGeneratorImage,
		Command: []string{
			"sh",
			"-c",
			`cd /home/ray/workspace/tls && openssl req -nodes -newkey rsa:2048 -keyout server.key -out server.csr -subj '/CN=ray-head' && printf "authorityKeyIdentifier=keyid,issuer\nbasicConstraints=CA:FALSE\nsubjectAltName = @alt_names\n[alt_names]\nDNS.1 = 127.0.0.1\nDNS.2 = localhost\nDNS.3 = ${FQ_RAY_IP}\nDNS.4 = $(awk 'END{print $1}' /etc/hosts)">./domain.ext && cp /home/ray/workspace/ca/* . && openssl x509 -req -CA ca.crt -CAkey ca.key -in server.csr -out server.crt -days 365 -CAcreateserial -extfile domain.ext`,
		},
		VolumeMounts: certVolumeMounts(),
	}
	return initContainerWorker
}

func validateHeadInitContainers(rayCluster *rayv1.RayCluster, cfg *config.KubeRayConfiguration) field.ErrorList {
	var allErrors field.ErrorList

	if err := contains(rayCluster.Spec.HeadGroupSpec.Template.Spec.InitContainers, rayHeadInitContainer(rayCluster, cfg), byContainerName,
		field.NewPath("spec", "headGroupSpec", "template", "spec", "initContainers"),
		"create-cert Init Container is immutable"); err != nil {
		allErrors = append(allErrors, err)
	}

	return allErrors
}

func validateWorkerInitContainers(rayCluster *rayv1.RayCluster, cfg *config.KubeRayConfiguration) field.ErrorList {
	var allErrors field.ErrorList

	for i := range rayCluster.Spec.WorkerGroupSpecs {
		workerSpec := &rayCluster.Spec.WorkerGroupSpecs[i]
		if err := contains(workerSpec.Template.Spec.InitContainers, rayWorkerInitContainer(cfg), byContainerName,
			field.NewPath("spec", "workerGroupSpecs", strconv.Itoa(i), "template", "spec", "initContainers"),
			"create-cert Init Container is immutable"); err != nil {
			allErrors = append(allErrors, err)
		}
	}

	return allErrors
}

func validateHeadEnvVars(rayCluster *rayv1.RayCluster) field.ErrorList {
	var allErrors field.ErrorList

	for _, envVar := range envVarList() {
		if err := contains(rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].Env, envVar, byEnvVarName,
			field.NewPath("spec", "headGroupSpec", "template", "spec", "containers", strconv.Itoa(0), "env"),
			"RAY_TLS related environment variables are immutable"); err != nil {
			allErrors = append(allErrors, err)
		}
	}

	return allErrors
}

func validateWorkerEnvVars(rayCluster *rayv1.RayCluster) field.ErrorList {
	var allErrors field.ErrorList

	for i := range rayCluster.Spec.WorkerGroupSpecs {
		workerSpec := &rayCluster.Spec.WorkerGroupSpecs[i]
		for _, envVar := range envVarList() {
			if err := contains(workerSpec.Template.Spec.Containers[0].Env, envVar, byEnvVarName,
				field.NewPath("spec", "workerGroupSpecs", strconv.Itoa(i), "template", "spec", "containers", strconv.Itoa(0), "env"),
				"RAY_TLS related environment variables are immutable"); err != nil {
				allErrors = append(allErrors, err)
			}
		}
	}

	return allErrors
}

func certVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "ca-vol",
			MountPath: "/home/ray/workspace/ca",
			ReadOnly:  true,
		},
		{
			Name:      "server-cert",
			MountPath: "/home/ray/workspace/tls",
			ReadOnly:  false,
		},
	}
}
