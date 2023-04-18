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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EIPAssociationSpec struct {
	PodName string `json:"podName,omitempty"`
	EIPName string `json:"eipName,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Pod Name",type=string,JSONPath=`.spec.podName`
// +kubebuilder:printcolumn:name="EIP Name",type=string,JSONPath=`.spec.eipName`
type EIPAssociation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec EIPAssociationSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type EIPAssociationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EIPAssociation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EIPAssociation{}, &EIPAssociationList{})
}
