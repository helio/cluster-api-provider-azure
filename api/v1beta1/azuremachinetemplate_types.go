/*
Copyright 2021 The Kubernetes Authors.

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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// AzureMachineTemplateStatus defines the observed state for an AzureMachineTemplate.
type AzureMachineTemplateStatus struct {
	// Capacity defines the resource capacity for this machine.
	// This value is used for autoscaling from zero operations as defined in:
	// https://github.com/kubernetes-sigs/cluster-api/blob/main/docs/proposals/20210310-opt-in-autoscaling-from-zero.md
	// +optional
	Capacity corev1.ResourceList `json:"capacity,omitempty"`
}

// AzureMachineTemplateSpec defines the desired state of AzureMachineTemplate.
type AzureMachineTemplateSpec struct {
	Template AzureMachineTemplateResource `json:"template"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=azuremachinetemplates,scope=Namespaced,categories=cluster-api
// +kubebuilder:storageversion

// AzureMachineTemplate is the Schema for the azuremachinetemplates API.
type AzureMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureMachineTemplateSpec   `json:"spec,omitempty"`
	Status AzureMachineTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AzureMachineTemplateList contains a list of AzureMachineTemplates.
type AzureMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureMachineTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AzureMachineTemplate{}, &AzureMachineTemplateList{})
}

// AzureMachineTemplateResource describes the data needed to create an AzureMachine from a template.
type AzureMachineTemplateResource struct {
	// +optional
	ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the specification of the desired behavior of the machine.
	Spec AzureMachineSpec `json:"spec"`
}
