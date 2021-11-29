/*
Copyright 2021.

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

package v1

import (
	"fmt"

	"github.com/spf13/cast"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ElasticWebSpec defines the desired state of ElasticWeb
type ElasticWebSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ElasticWeb. Edit elasticweb_types.go to remove/update
	// Foo string `json:"foo,omitempty"`
	// 镜像
	Image string `json:"image"`
	// 端口
	Port *int `json:"port"`
	// 单个 pod 的 qps
	SingleQPS *int `json:"singleQPS"`
	// 总 qps
	TotalQPS *int `json:"totalQPS"`
}

// ElasticWebStatus defines the observed state of ElasticWeb
type ElasticWebStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// 实际 qps
	RealQPS *int `json:"realQPS"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ElasticWeb is the Schema for the elasticwebs API
type ElasticWeb struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElasticWebSpec   `json:"spec,omitempty"`
	Status ElasticWebStatus `json:"status,omitempty"`
}

func (web *ElasticWeb) String() string {
	realQPS := ""
	if web.Status.RealQPS == nil {
		realQPS = "nil"
	} else {
		realQPS = cast.ToString(*web.Status.RealQPS)
	}

	return fmt.Sprintf("Image [%s], Port [%d], SingleQPS [%d], TotalQPS [%d], RealQPS [%s]", web.Spec.Image, *web.Spec.Port, *web.Spec.SingleQPS, *web.Spec.TotalQPS, realQPS)
}

//+kubebuilder:object:root=true

// ElasticWebList contains a list of ElasticWeb
type ElasticWebList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElasticWeb `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ElasticWeb{}, &ElasticWebList{})
}
