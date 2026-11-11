package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CheckRunSpec requests a validation of an instance.
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="spec is immutable"
type CheckRunSpec struct {
	// Instance is the name of the ChallengeInstance to check (same namespace).
	Instance string `json:"instance"`
	// Tasks to check. Empty checks every task.
	// +optional
	Tasks []string `json:"tasks,omitempty"`
}

// CheckRunPhase is the lifecycle phase of a check run.
// +kubebuilder:validation:Enum=Pending;Running;Completed;Error
type CheckRunPhase string

const (
	// CheckRunPending waits for the checker to start.
	CheckRunPending CheckRunPhase = "Pending"
	// CheckRunRunning is being executed by the checker.
	CheckRunRunning CheckRunPhase = "Running"
	// CheckRunCompleted ran to completion: see results (tasks may have failed).
	CheckRunCompleted CheckRunPhase = "Completed"
	// CheckRunError could not run (checker crash, invalid output, timeout).
	// It says nothing about the tasks.
	CheckRunError CheckRunPhase = "Error"
)

// CheckRunStatus reports the outcome of a check run.
type CheckRunStatus struct {
	// Phase of the run.
	// +optional
	Phase CheckRunPhase `json:"phase,omitempty"`
	// StartedAt is when the checker started.
	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`
	// CompletedAt is when the run completed or errored.
	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`
	// Results per task.
	// +optional
	// +listType=map
	// +listMapKey=task
	Results []TaskResult `json:"results,omitempty"`
	// Message explains an error.
	// +optional
	Message string `json:"message,omitempty"`
}

// CheckRun is one validation of a ChallengeInstance.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Instance",type=string,JSONPath=`.spec.instance`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type CheckRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CheckRunSpec `json:"spec"`
	// +optional
	Status CheckRunStatus `json:"status,omitempty"`
}

// CheckRunList is a list of CheckRun.
// +kubebuilder:object:root=true
type CheckRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CheckRun `json:"items"`
}
