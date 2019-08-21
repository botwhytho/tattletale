/*

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
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SharedConfigMapSpec defines the desired state of SharedConfigMap
type SharedConfigMapSpec struct {
	// The name of the source configmap to be shared
	SourceConfigMap string `json:"sourceConfigMap"`

	// The namespace of the source configmap to be shared
	SourceNamespace string `json:"sourceNamespace"`

	// The sync period (default: 30s)
	// +optional
	SyncPeriod string `json:"syncPeriod,omitempty"`

	// The list of target namespaces to sync to
	TargetNamespaces []string `json:"targetNamespaces"`
}

// SharedConfigMapStatus defines the observed state of SharedConfigMap
type SharedConfigMapStatus struct {
	// The status of the source configmap to be shared
	SourceConfigMap string `json:"sourceConfigMap"`

	// The status of target configmap to be synched
	TargetConfigMaps []string `json:"targetConfigMaps"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SharedConfigMap is the Schema for the sharedconfigmaps API
type SharedConfigMap struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SharedConfigMapSpec   `json:"spec,omitempty"`
	Status SharedConfigMapStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SharedConfigMapList contains a list of SharedConfigMap
type SharedConfigMapList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SharedConfigMap `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SharedConfigMap{}, &SharedConfigMapList{})
}
