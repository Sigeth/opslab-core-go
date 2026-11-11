package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PlatformConfigName is the name of the only allowed PlatformConfig.
const PlatformConfigName = "default"

// PlatformConfigSpec is the cluster-wide policy for challenges and instances.
type PlatformConfigSpec struct {
	// AllowedRuntimes restricts the runtimes challenges may use. Empty allows
	// every runtime the cluster supports.
	// +optional
	AllowedRuntimes []Runtime `json:"allowedRuntimes,omitempty"`
	// MaxInstanceLifetime caps the lifetime of any instance, extensions included.
	// +optional
	MaxInstanceLifetime *metav1.Duration `json:"maxInstanceLifetime,omitempty"`
	// AllowedEgress is the most permissive egress a challenge may request.
	// Defaults to "none".
	// +optional
	AllowedEgress EgressMode `json:"allowedEgress,omitempty"`
	// RegistryMirror is the in-cluster registry installed challenges and
	// their images are mirrored to, e.g. "zot.opslab-system.svc:5000".
	// +optional
	RegistryMirror string `json:"registryMirror,omitempty"`
}

// RuntimeCapability reports whether a runtime can run on the cluster.
type RuntimeCapability struct {
	// Runtime this capability is about.
	Runtime Runtime `json:"runtime"`
	// Available is true when instances using the runtime can be scheduled.
	Available bool `json:"available"`
	// Reason explains why the runtime is unavailable, e.g. "no node exposes /dev/kvm".
	// +optional
	Reason string `json:"reason,omitempty"`
}

// Capacity summarizes cluster resources usable by instances.
type Capacity struct {
	// AllocatableCPU on the nodes that can run instances.
	AllocatableCPU resource.Quantity `json:"allocatableCPU"`
	// AllocatableMemory on the nodes that can run instances.
	AllocatableMemory resource.Quantity `json:"allocatableMemory"`
	// RequestedCPU by running instances.
	RequestedCPU resource.Quantity `json:"requestedCPU"`
	// RequestedMemory by running instances.
	RequestedMemory resource.Quantity `json:"requestedMemory"`
}

// PlatformConfigStatus reports what the cluster can run.
type PlatformConfigStatus struct {
	// ObservedGeneration is the generation last processed by the operator.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Runtimes reports the availability of each runtime.
	// +optional
	// +listType=map
	// +listMapKey=runtime
	Runtimes []RuntimeCapability `json:"runtimes,omitempty"`
	// KVMNodes is the number of nodes able to run virtual machines.
	// +optional
	KVMNodes int32 `json:"kvmNodes,omitempty"`
	// Capacity of the cluster for instances.
	// +optional
	Capacity *Capacity `json:"capacity,omitempty"`
	// Conditions of the platform.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// PlatformConfig is the cluster-wide configuration of opslab. Only one, named
// "default", may exist.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:validation:XValidation:rule="self.metadata.name == 'default'",message="the PlatformConfig must be named 'default'"
// +kubebuilder:printcolumn:name="KVM nodes",type=integer,JSONPath=`.status.kvmNodes`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type PlatformConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec PlatformConfigSpec `json:"spec,omitempty"`
	// +optional
	Status PlatformConfigStatus `json:"status,omitempty"`
}

// PlatformConfigList is a list of PlatformConfig.
// +kubebuilder:object:root=true
type PlatformConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PlatformConfig `json:"items"`
}
