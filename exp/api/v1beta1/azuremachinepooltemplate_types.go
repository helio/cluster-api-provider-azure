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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

type (
	// AzureMachinePoolTemplateResource describes the data needed to create a AzureMachinePool from a template.
	AzureMachinePoolTemplateResource struct {
		// Standard object's metadata.
		// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
		// +optional
		ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty"`

		Spec AzureMachinePoolSpec `json:"spec,omitempty"`
	}

	AzureMachinePoolTemplateSpec struct {
		Template AzureMachinePoolTemplateResource `json:"template"`
	}

	// +kubebuilder:object:root=true
	// +kubebuilder:resource:path=azuremachinepooltemplates,scope=Namespaced,categories=cluster-api,shortName=ampt
	// +kubebuilder:storageversion
	// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of AzureMachinePoolTemplate"

	// AzureMachinePoolTemplate is the Schema for the azuremachinepooltemplates API.
	AzureMachinePoolTemplate struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata,omitempty"`

		Spec AzureMachinePoolTemplateSpec `json:"spec,omitempty"`
	}

	// +kubebuilder:object:root=true

	// AzureMachinePoolTemplateList contains a list of AzureMachinePoolTemplate.
	AzureMachinePoolTemplateList struct {
		metav1.TypeMeta `json:",inline"`
		metav1.ListMeta `json:"metadata,omitempty"`
		Items           []AzureMachinePoolTemplate `json:"items"`
	}
)

func init() {
	SchemeBuilder.Register(&AzureMachinePoolTemplate{}, &AzureMachinePoolTemplateList{})
}
