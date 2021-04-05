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
	"k8s.io/apimachinery/pkg/types"
)

type EIPAssignment struct {
	// +kubebuilder:validation:MinLength=0
	// +optional
	PodName                  string `json:"podName,omitempty"`
	PrivateIPAddress         string `json:"privateIPAddress,omitempty"`
	ENI                      string `json:"eni,omitempty"`
	ENIPrivateIPAddressIndex int    `json:"eniPrivateIPAddressIndex,omitempty"`

	//ElasticNetworkInterface EIPElasticNetworkInterfaceAssignment `json:"elasticNetworkInterface,omitempty"`
	//NetworkLoadBalancer     EIPNetworkLoadBalancerAssignment `json:"networkLoadBalancer,omitempty"`
}

// EIPSpec defines the desired state of EIP
type EIPSpec struct {
	// Which resource this EIP should be assigned to.
	//
	// If not given, it will not be assigned to anything.
	//
	// +optional
	Assignment *EIPAssignment `json:"assignment,omitempty"`

	PublicIPv4Pool  string `json:"publicIPv4Pool,omitempty"`
	PublicIPAddress string `json:"publicIPAddress,omitempty"`

	// Tags that will be applied to the created EIP.
	// +optional
	Tags *map[string]string `json:"tags,omitempty"`
}

// EIPStatus defines the observed state of EIP
type EIPStatus struct {
	// Current state of the EIP object.
	//
	// State transfer diagram:
	//
	//                   /------- unassigning <----\--------------\
	//                   |                         |              |
	//  *start*:         V                         |              |
	// allocating -> allocated <-> assigning -> assigned <-> reassigning
	//                   |             |
	//   *end*:          |             |
	//  releasing <------/-------------/
	State string `json:"state"`

	AllocationId    string `json:"allocationId,omitempty"`
	PublicIPAddress string `json:"publicIPAddress,omitempty"`

	AssociationId string         `json:"associationId,omitempty"`
	Assignment    *EIPAssignment `json:"assignment,omitempty"`
	PodUID        types.UID      `json:"podUUID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Public IP",type=string,JSONPath=`.status.publicIPAddress`
// +kubebuilder:printcolumn:name="Private IP",type=string,JSONPath=`.status.assignment.privateIPAddress`
// +kubebuilder:printcolumn:name="Pod",type=string,JSONPath=`.status.assignment.podName`
// +kubebuilder:printcolumn:name="ENI",type=string,JSONPath=`.status.assignment.eni`

// EIP is the Schema for the eips API
type EIP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EIPSpec   `json:"spec,omitempty"`
	Status EIPStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EIPList contains a list of EIP
type EIPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EIP `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EIP{}, &EIPList{})
}
