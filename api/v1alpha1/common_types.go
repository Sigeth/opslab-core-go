package v1alpha1

// Runtime is the isolation technology a machine runs on.
// +kubebuilder:validation:Enum=pod;sysbox;vcluster;vm
type Runtime string

const (
	// RuntimePod is a plain pod under the "restricted" Pod Security Standard.
	RuntimePod Runtime = "pod"
	// RuntimeSysbox is a system container (Sysbox runtime class): systemd and
	// container engines inside, without a privileged pod.
	RuntimeSysbox Runtime = "sysbox"
	// RuntimeVCluster is a virtual Kubernetes cluster in the instance namespace.
	RuntimeVCluster Runtime = "vcluster"
	// RuntimeVM is a KubeVirt virtual machine.
	RuntimeVM Runtime = "vm"
)

// Runtimes lists every runtime, in increasing isolation order.
var Runtimes = []Runtime{RuntimePod, RuntimeSysbox, RuntimeVCluster, RuntimeVM}

// Category is the topic of a challenge.
// +kubebuilder:validation:Enum=linux-hardening;containers;kubernetes
type Category string

const (
	CategoryLinuxHardening Category = "linux-hardening"
	CategoryContainers     Category = "containers"
	CategoryKubernetes     Category = "kubernetes"
)

// Categories lists every category.
var Categories = []Category{CategoryLinuxHardening, CategoryContainers, CategoryKubernetes}

// Difficulty is the expected difficulty of a challenge.
// +kubebuilder:validation:Enum=easy;medium;hard;insane
type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
	DifficultyInsane Difficulty = "insane"
)

// Difficulties lists every difficulty, easiest first.
var Difficulties = []Difficulty{DifficultyEasy, DifficultyMedium, DifficultyHard, DifficultyInsane}

// ConditionReady is the condition type every opslab resource reports.
const ConditionReady = "Ready"

// TaskResult is the outcome of checking one task. Checkers print one JSON
// TaskResult per line on stdout; the operator copies them into CheckRun status.
type TaskResult struct {
	// Task is the task ID.
	Task string `json:"task"`
	// Passed is true when the task is solved.
	Passed bool `json:"passed"`
	// Message is shown to the student (why it failed, or a confirmation).
	// +optional
	Message string `json:"message,omitempty"`
}
