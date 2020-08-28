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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ENIAttachment struct {
	// +kubebuilder:validation:MinLength=0
	PodName string `json:"podName,omitempty"`
}

// ENISpec defines the desired state of an ElasticNetworkInterface
type ENISpec struct {
	SubnetID                       string   `json:"subnetID"`
	SecurityGroups                 []string `json:"securityGroups"`
	SecondaryPrivateIPAddressCount int64    `json:"secondaryPrivateIPAddressCount,omitempty"`

	// +optional
	Attachment *ENIAttachment `json:"attachment,omitempty"`

	Description string `json:"description,omitempty"`
}

// ENIStatus defines the observed state of ENI
type ENIStatus struct {
	NetworkInterfaceID string `json:"networkInterfaceID"`
	MacAddress         string `json:"macAddress"`

	// +optional
	PrivateIPAddresses []string       `json:"privateIPAddresses,omitempty"`
	Attachment         *ENIAttachment `json:"attachment,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Pod",type=string,JSONPath=`.status.attachment.podName`
// +kubebuilder:printcolumn:name="Private IP addresses",type=string,JSONPath=`.status.privateIPAddresses`

// ENI is the Schema for the enis API
type ENI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ENISpec   `json:"spec,omitempty"`
	Status ENIStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ENIList contains a list of ENI
type ENIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ENI `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ENI{}, &ENIList{})
}
