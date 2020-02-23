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

// Stores the namespace of a target and an optional 'NewName' if the secret will be renamed in the target namespace
type Target struct {
	Namespace string `json:"namespace"`
	NewName   string `json:"newName,omitempty"`
}

// SharedSecretSpec defines the desired state of SharedSecret
type SharedSecretSpec struct {
	// The name of the source secret to be shared
	SourceSecret string `json:"sourceSecret"`

	// The namespace of the source secret to be shared
	SourceNamespace string `json:"sourceNamespace"`

	// The list of target namespaces to sync to
	Targets []Target `json:"targets"`
}

// SharedSecretStatus defines the observed state of SharedSecret
type SharedSecretStatus struct {
	// The status of the source secret to be shared
	SourceSecret string `json:"sourceSecret"`

	// The status of target secrets to be synched
	TargetSecrets []string `json:"targetSecrets"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SharedSecret is the Schema for the sharedsecrets API
type SharedSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SharedSecretSpec   `json:"spec,omitempty"`
	Status SharedSecretStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SharedSecretList contains a list of SharedSecret
type SharedSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SharedSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SharedSecret{}, &SharedSecretList{})
}
