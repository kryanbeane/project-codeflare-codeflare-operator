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

package config

import (
	awconfig "github.com/project-codeflare/appwrapper/pkg/config"

	configv1alpha1 "k8s.io/component-base/config/v1alpha1"
)

type CodeFlareOperatorConfiguration struct {
	// ClientConnection provides additional configuration options for Kubernetes
	// API server client.
	ClientConnection *ClientConnection `json:"clientConnection,omitempty"`

	// ControllerManager returns the configurations for controllers
	ControllerManager `json:",inline"`

	KubeRay *KubeRayConfiguration `json:"kuberay,omitempty"`

	AppWrapper *AppWrapperConfiguration `json:"appwrapper,omitempty"`
}

type AppWrapperConfiguration struct {
	// Enabled controls whether or not the AppWrapper Controller is enabled
	Enabled *bool `json:"enabled,omitempty"`

	// AppWrapper contains the AppWrapper controller configuration
	// +optional
	Config *awconfig.AppWrapperConfig `json:",inline"`
}

type KubeRayConfiguration struct {
	// Ingress domain for OpenShift Route an Ingress hostnames
	IngressDomain string `json:"ingressDomain,omitempty"`
	// Configuration for enabling mTLS
	MTLSEnabled *bool `json:"mtlsEnabled,omitempty"`
	// EnableIstio determines whether Istio is enabled in the cluster.
	// If true, the operator will configure Kiali and Jaeger to monitor Ray clusters.
	EnableIstio *bool `json:"enableIstio,omitempty"`
}

type ControllerManager struct {
	// Metrics contains the controller metrics configuration
	// +optional
	Metrics MetricsConfiguration `