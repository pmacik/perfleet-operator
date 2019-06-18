package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PodFleetSpec defines the desired state of PodFleet
// +k8s:openapi-gen=true
type PodFleetSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Workers int32 `json:"workers"`
}

// PodFleetStatus defines the observed state of PodFleet
// +k8s:openapi-gen=true
type PodFleetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Started        bool  `json:"started"`
	WarmingUp      bool  `json:"warmingUp"`
	WorkersPending int32 `json:"workersPending"`
	WorkersWorking int32 `json:"workersWorking"`
	WorkersDone    int32 `json:"workersDone"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PodFleet is the Schema for the podfleets API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type PodFleet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodFleetSpec   `json:"spec,omitempty"`
	Status PodFleetStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PodFleetList contains a list of PodFleet
type PodFleetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodFleet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PodFleet{}, &PodFleetList{})
}
