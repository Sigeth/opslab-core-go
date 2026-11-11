package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstanceOwner identifies the platform user an instance belongs to.
type InstanceOwner struct {
	// UserID is the platform user ID.
	UserID string `json:"userID"`
	// DisplayName of the user, for operators reading kubectl output.
	// +optional
	DisplayName string `json:"displayName,omitempty"`
}

// ChallengeInstanceSpec requests a running instance of a challenge.
type ChallengeInstanceSpec struct {
	// Challenge is the name of the installed Challenge.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="challenge is immutable"
	Challenge string `json:"challenge"`
	// Owner of the instance.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="owner is immutable"
	Owner InstanceOwner `json:"owner"`
	// ExpiresAt is when the instance is torn down. The platform moves it
	// forward to extend the instance. Defaults to creation time plus the
	// challenge's default lifetime; capped by the challenge's and the
	// platform's maximum lifetime.
	// +optional
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`
}

// InstancePhase is the lifecycle phase of an instance.
// +kubebuilder:validation:Enum=Queued;Provisioning;Ready;Expired;Failed
type InstancePhase string

const (
	// InstanceQueued waits for cluster capacity.
	InstanceQueued InstancePhase = "Queued"
	// InstanceProvisioning creates the machines and runs setup.
	InstanceProvisioning InstancePhase = "Provisioning"
	// InstanceReady can be used by the student.
	InstanceReady InstancePhase = "Ready"
	// InstanceExpired was torn down because it reached its expiry.
	InstanceExpired InstancePhase = "Expired"
	// InstanceFailed could not be provisioned.
	InstanceFailed InstancePhase = "Failed"
)

// MachineStatus reports one machine of an instance.
type MachineStatus struct {
	// Name of the machine.
	Name string `json:"name"`
	// Runtime of the machine.
	Runtime Runtime `json:"runtime"`
	// Ready is true when the machine is up and its setup completed.
	Ready bool `json:"ready"`
	// Address reachable from the gateway and the checker (IP or DNS name).
	// +optional
	Address string `json:"address,omitempty"`
	// Pod running the machine, in the instance namespace (pod and sysbox
	// runtimes). Terminals and checks exec into its MachineContainer.
	// +optional
	Pod string `json:"pod,omitempty"`
}

// MachineContainer is the name of the container running a pod or sysbox
// machine inside MachineStatus.Pod.
const MachineContainer = "machine"

// ChallengeInstanceStatus reports the state of an instance.
type ChallengeInstanceStatus struct {
	// ObservedGeneration is the generation last processed by the operator.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Phase of the instance.
	// +optional
	Phase InstancePhase `json:"phase,omitempty"`
	// Namespace holding the instance resources.
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// ChallengeDigest is the package digest the instance was created from.
	// It does not change when the challenge is updated.
	// +optional
	ChallengeDigest string `json:"challengeDigest,omitempty"`
	// Machines of the instance.
	// +optional
	// +listType=map
	// +listMapKey=name
	Machines []MachineStatus `json:"machines,omitempty"`
	// ReadyAt is when the instance first became ready.
	// +optional
	ReadyAt *metav1.Time `json:"readyAt,omitempty"`
	// ExpiresAt is the effective expiry, after defaults and caps.
	// +optional
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`
	// Message explains the current phase (queue position, failure reason...).
	// +optional
	Message string `json:"message,omitempty"`
	// Conditions of the instance.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ChallengeInstance is one student's running attempt at a challenge.
// Deleting it tears the environment down; everything in it is lost.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ci
// +kubebuilder:printcolumn:name="Challenge",type=string,JSONPath=`.spec.challenge`
// +kubebuilder:printcolumn:name="Owner",type=string,JSONPath=`.spec.owner.displayName`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Expires",type=date,JSONPath=`.status.expiresAt`
type ChallengeInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ChallengeInstanceSpec `json:"spec"`
	// +optional
	Status ChallengeInstanceStatus `json:"status,omitempty"`
}

// ChallengeInstanceList is a list of ChallengeInstance.
// +kubebuilder:object:root=true
type ChallengeInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ChallengeInstance `json:"items"`
}
