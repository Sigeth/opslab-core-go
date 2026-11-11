package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ChallengePackageKind is the kind of a challenge manifest.
const ChallengePackageKind = "ChallengePackage"

// ChallengePackage is the manifest of a challenge (challenge.yaml). It is not a
// Kubernetes resource: it ships inside a challenge OCI artifact and is embedded
// in Challenge objects once installed.
type ChallengePackage struct {
	// APIVersion must be "opslab.io/v1alpha1".
	APIVersion string `json:"apiVersion"`
	// Kind must be "ChallengePackage".
	Kind string `json:"kind"`
	// Metadata identifies the package.
	Metadata PackageMetadata `json:"metadata"`
	// Spec describes the challenge.
	Spec ChallengePackageSpec `json:"spec"`
}

// PackageMetadata identifies a challenge package.
type PackageMetadata struct {
	// Name is the package name: lowercase letters, digits and '-' (DNS-1123 label).
	Name string `json:"name"`
	// Version is a semantic version (https://semver.org), e.g. "1.2.0".
	Version string `json:"version"`
}

// ChallengePackageSpec describes a challenge: what it is about, the machines
// it runs on, how students access them, and the tasks to solve.
type ChallengePackageSpec struct {
	// Title is the human-readable name of the challenge.
	Title string `json:"title"`
	// Summary is a one or two sentence pitch shown in the catalog.
	Summary string `json:"summary"`
	// Category is the topic of the challenge.
	Category Category `json:"category"`
	// Difficulty is the expected difficulty.
	Difficulty Difficulty `json:"difficulty"`
	// EstimatedTime is how long a student typically needs, e.g. "30m".
	// +optional
	EstimatedTime *metav1.Duration `json:"estimatedTime,omitempty"`
	// Tags are free-form keywords used for search.
	// +optional
	Tags []string `json:"tags,omitempty"`
	// Authors of the challenge.
	// +optional
	Authors []Author `json:"authors,omitempty"`
	// License is the SPDX identifier of the challenge license, e.g. "MIT".
	// +optional
	License string `json:"license,omitempty"`
	// Requires lists compatibility requirements.
	// +optional
	Requires *Requirements `json:"requires,omitempty"`
	// Permissions declares what the challenge needs from the cluster. They are
	// shown to the admin at install time and checked against cluster policy.
	// +optional
	Permissions *Permissions `json:"permissions,omitempty"`
	// Lifetime bounds how long an instance lives.
	// +optional
	Lifetime *Lifetime `json:"lifetime,omitempty"`
	// Machines that make up one instance of the challenge.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	Machines []Machine `json:"machines"`
	// ExtraManifests are paths (relative to the package root) of Kubernetes
	// manifests applied in the instance namespace. Only namespaced resources
	// are allowed. Prefer machines: use this as an escape hatch.
	// +optional
	ExtraManifests []string `json:"extraManifests,omitempty"`
	// Access defines how the student reaches the machines.
	Access Access `json:"access"`
	// Validation configures how tasks are checked.
	// +optional
	Validation *Validation `json:"validation,omitempty"`
	// Tasks are the gradable objectives of the challenge.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=64
	Tasks []Task `json:"tasks"`
}

// Author of a challenge.
type Author struct {
	// Name of the author.
	Name string `json:"name"`
	// URL of the author (website, GitHub profile...).
	// +optional
	URL string `json:"url,omitempty"`
}

// Requirements lists compatibility requirements of a package.
type Requirements struct {
	// Platform is a semantic version constraint on the platform version, e.g. ">=0.3.0".
	// +optional
	Platform string `json:"platform,omitempty"`
}

// EgressMode controls outbound network access from an instance.
// +kubebuilder:validation:Enum=none;internet;allowlist
type EgressMode string

const (
	// EgressNone blocks all outbound traffic leaving the instance.
	EgressNone EgressMode = "none"
	// EgressInternet allows outbound traffic to the internet.
	EgressInternet EgressMode = "internet"
	// EgressAllowlist allows outbound traffic to the listed hosts only.
	EgressAllowlist EgressMode = "allowlist"
)

// EgressModes lists every egress mode.
var EgressModes = []EgressMode{EgressNone, EgressInternet, EgressAllowlist}

// Permissions declares what a challenge needs from the cluster.
type Permissions struct {
	// Egress is the outbound network access the instance needs. Defaults to "none".
	// +optional
	Egress EgressMode `json:"egress,omitempty"`
	// EgressAllowlist lists the host names reachable when egress is "allowlist".
	// +optional
	EgressAllowlist []string `json:"egressAllowlist,omitempty"`
}

// Lifetime bounds how long an instance lives.
type Lifetime struct {
	// Default lifetime of a new instance. Defaults to 1h.
	// +optional
	Default *metav1.Duration `json:"default,omitempty"`
	// Max is the maximum lifetime, extensions included. Defaults to 3h.
	// +optional
	Max *metav1.Duration `json:"max,omitempty"`
}

// Machine is one unit of an instance (a pod, system container, virtual
// cluster or virtual machine).
type Machine struct {
	// Name of the machine, unique in the package (DNS-1123 label).
	Name string `json:"name"`
	// Runtime is the isolation technology the machine runs on.
	Runtime Runtime `json:"runtime"`
	// Image is a container image (pod, sysbox) or a KubeVirt containerDisk
	// image (vm). Pin it by digest. Not used by vcluster.
	// +optional
	Image string `json:"image,omitempty"`
	// Resources requested by the machine.
	// +optional
	Resources *MachineResources `json:"resources,omitempty"`
	// Setup lists scripts (paths relative to the package root) run once as
	// root at provisioning, in order. Not supported by vcluster: use seed.
	// +optional
	Setup []string `json:"setup,omitempty"`
	// Kubernetes is the version of the virtual cluster, e.g. "1.34" (vcluster only).
	// +optional
	Kubernetes string `json:"kubernetes,omitempty"`
	// Seed lists manifest files or directories (paths relative to the package
	// root) applied into the virtual cluster at provisioning (vcluster only).
	// +optional
	Seed []string `json:"seed,omitempty"`
	// KubeconfigFor injects into this machine a kubeconfig for the named
	// vcluster machine.
	// +optional
	KubeconfigFor string `json:"kubeconfigFor,omitempty"`
}

// MachineResources are the resources requested by a machine.
type MachineResources struct {
	// CPU, e.g. "500m" or "1".
	CPU resource.Quantity `json:"cpu"`
	// Memory, e.g. "512Mi" or "1Gi".
	Memory resource.Quantity `json:"memory"`
}

// Access defines how the student reaches the machines.
type Access struct {
	// Terminals offered to the student, in display order.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	Terminals []Terminal `json:"terminals"`
	// Web endpoints inside the instance proxied to the student's browser.
	// +optional
	// +kubebuilder:validation:MaxItems=16
	Web []WebEndpoint `json:"web,omitempty"`
	// IDE runs a browser IDE (openvscode-server) on a machine.
	// +optional
	IDE *IDE `json:"ide,omitempty"`
}

// Terminal is a shell offered to the student.
type Terminal struct {
	// Machine the terminal opens on. Cannot be a vcluster machine.
	Machine string `json:"machine"`
	// User the shell runs as. Defaults to "student".
	// +optional
	User string `json:"user,omitempty"`
	// Label shown on the terminal tab. Defaults to the machine name.
	// +optional
	Label string `json:"label,omitempty"`
}

// WebEndpoint is an HTTP service inside the instance shown to the student.
type WebEndpoint struct {
	// Machine serving the endpoint.
	Machine string `json:"machine"`
	// Port the service listens on.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`
	// Label shown in the UI.
	// +optional
	Label string `json:"label,omitempty"`
}

// IDE configures the browser IDE.
type IDE struct {
	// Machine the IDE runs on. Cannot be a vcluster machine.
	Machine string `json:"machine"`
}

// ValidationMode controls when tasks are checked.
// +kubebuilder:validation:Enum=on-demand;auto
type ValidationMode string

const (
	// ValidationOnDemand checks tasks when the student asks for it.
	ValidationOnDemand ValidationMode = "on-demand"
	// ValidationAuto also checks tasks periodically while the instance is ready.
	ValidationAuto ValidationMode = "auto"
)

// ValidationModes lists every validation mode.
var ValidationModes = []ValidationMode{ValidationOnDemand, ValidationAuto}

// Validation configures how tasks are checked.
type Validation struct {
	// Mode controls when tasks are checked. Defaults to "on-demand".
	// +optional
	Mode ValidationMode `json:"mode,omitempty"`
	// Interval between automatic checks (auto mode only). Defaults to 30s, minimum 10s.
	// +optional
	Interval *metav1.Duration `json:"interval,omitempty"`
	// CheckerImage is a custom checker image. When empty, the built-in checker
	// evaluates each task's check.
	// +optional
	CheckerImage string `json:"checkerImage,omitempty"`
}

// Task is a gradable objective.
type Task struct {
	// ID of the task, unique in the package (DNS-1123 label).
	ID string `json:"id"`
	// Title shown to the student.
	Title string `json:"title"`
	// Points awarded when the task is solved.
	// +kubebuilder:validation:Minimum=0
	Points int32 `json:"points"`
	// Hints the student can reveal, in order.
	// +optional
	// +kubebuilder:validation:MaxItems=8
	Hints []Hint `json:"hints,omitempty"`
	// Check evaluated by the built-in checker. Required unless the package
	// uses a custom checker image.
	// +optional
	Check *Check `json:"check,omitempty"`
}

// Hint is a clue a student can reveal.
type Hint struct {
	// Text of the hint (markdown).
	Text string `json:"text"`
	// Cost in points deducted when the hint is revealed.
	// +kubebuilder:validation:Minimum=0
	// +optional
	Cost int32 `json:"cost,omitempty"`
}

// Check is a built-in check. Exactly one check type must be set.
// +kubebuilder:validation:XValidation:rule="(has(self.shell) ? 1 : 0) + (has(self.file) ? 1 : 0) + (has(self.sshLogin) ? 1 : 0) + (has(self.port) ? 1 : 0) + (has(self.http) ? 1 : 0) + (has(self.kubeCEL) ? 1 : 0) + (has(self.goss) ? 1 : 0) == 1",message="exactly one check type must be set"
type Check struct {
	// Machine the check targets.
	Machine string `json:"machine"`
	// Shell runs a command on the machine.
	// +optional
	Shell *ShellCheck `json:"shell,omitempty"`
	// File asserts properties of a file on the machine.
	// +optional
	File *FileCheck `json:"file,omitempty"`
	// SSHLogin attempts an SSH login from the checker (vm and sysbox machines).
	// +optional
	SSHLogin *SSHLoginCheck `json:"sshLogin,omitempty"`
	// Port probes a port from the checker.
	// +optional
	Port *PortCheck `json:"port,omitempty"`
	// HTTP sends a request from the checker.
	// +optional
	HTTP *HTTPCheck `json:"http,omitempty"`
	// KubeCEL evaluates a CEL expression over objects of a vcluster machine.
	// +optional
	KubeCEL *KubeCELCheck `json:"kubeCEL,omitempty"`
	// Goss runs a goss spec on the machine.
	// +optional
	Goss *GossCheck `json:"goss,omitempty"`
}

// ShellCheck runs a command on a machine.
type ShellCheck struct {
	// Run is the command, executed with /bin/sh -c.
	Run string `json:"run"`
	// User running the command. Defaults to "root".
	// +optional
	User string `json:"user,omitempty"`
	// ExpectExit is the expected exit code. Defaults to 0.
	// +optional
	ExpectExit *int32 `json:"expectExit,omitempty"`
	// ExpectOutput is a regular expression stdout must match.
	// +optional
	ExpectOutput string `json:"expectOutput,omitempty"`
}

// FileCheck asserts properties of a file.
type FileCheck struct {
	// Path of the file (absolute).
	Path string `json:"path"`
	// Exists is whether the file must exist. Defaults to true.
	// +optional
	Exists *bool `json:"exists,omitempty"`
	// Mode is the expected octal permission mode, e.g. "0600".
	// +optional
	Mode string `json:"mode,omitempty"`
	// Owner is the expected owner, "user" or "user:group".
	// +optional
	Owner string `json:"owner,omitempty"`
	// Contains is a regular expression the content must match.
	// +optional
	Contains string `json:"contains,omitempty"`
	// NotContains is a regular expression the content must not match.
	// +optional
	NotContains string `json:"notContains,omitempty"`
}

// SSHLoginMethod is how the checker authenticates.
// +kubebuilder:validation:Enum=key;password
type SSHLoginMethod string

const (
	// SSHLoginKey uses the student's key provisioned by the platform.
	SSHLoginKey SSHLoginMethod = "key"
	// SSHLoginPassword uses a password.
	SSHLoginPassword SSHLoginMethod = "password"
)

// Expectation of an access probe.
// +kubebuilder:validation:Enum=allowed;denied
type AccessExpectation string

const (
	ExpectAllowed AccessExpectation = "allowed"
	ExpectDenied  AccessExpectation = "denied"
)

// SSHLoginCheck attempts an SSH login from the checker.
type SSHLoginCheck struct {
	// User to log in as.
	User string `json:"user"`
	// Method of authentication. Defaults to "key".
	// +optional
	Method SSHLoginMethod `json:"method,omitempty"`
	// Password used with the "password" method.
	// +optional
	Password string `json:"password,omitempty"`
	// Expect is whether the login must be allowed or denied.
	Expect AccessExpectation `json:"expect"`
}

// PortState is the expected state of a port.
// +kubebuilder:validation:Enum=open;closed
type PortState string

const (
	PortOpen   PortState = "open"
	PortClosed PortState = "closed"
)

// Protocol of a port probe.
// +kubebuilder:validation:Enum=tcp;udp
type Protocol string

const (
	ProtocolTCP Protocol = "tcp"
	ProtocolUDP Protocol = "udp"
)

// PortCheck probes a port from the checker.
type PortCheck struct {
	// Port to probe.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`
	// Protocol of the probe. Defaults to "tcp".
	// +optional
	Protocol Protocol `json:"protocol,omitempty"`
	// Expect is the expected state of the port.
	Expect PortState `json:"expect"`
}

// HTTPCheck sends an HTTP request from the checker.
type HTTPCheck struct {
	// Port of the service.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`
	// Path of the request. Defaults to "/".
	// +optional
	Path string `json:"path,omitempty"`
	// Scheme is "http" or "https". Defaults to "http".
	// +kubebuilder:validation:Enum=http;https
	// +optional
	Scheme string `json:"scheme,omitempty"`
	// ExpectStatus is the expected status code. Defaults to 200.
	// +optional
	ExpectStatus int32 `json:"expectStatus,omitempty"`
	// ExpectBody is a regular expression the body must match.
	// +optional
	ExpectBody string `json:"expectBody,omitempty"`
}

// KubeCELCheck evaluates a CEL expression on Kubernetes objects. Exactly one
// of all, any and none must be set. The expression sees the object as `object`.
// +kubebuilder:validation:XValidation:rule="(has(self.all) ? 1 : 0) + (has(self.any) ? 1 : 0) + (has(self.none) ? 1 : 0) == 1",message="exactly one of all, any and none must be set"
type KubeCELCheck struct {
	// Resource selects the objects to evaluate.
	Resource ResourceSelector `json:"resource"`
	// All passes when the expression is true for every selected object (and at least one is selected).
	// +optional
	All string `json:"all,omitempty"`
	// Any passes when the expression is true for at least one selected object.
	// +optional
	Any string `json:"any,omitempty"`
	// None passes when the expression is false for every selected object.
	// +optional
	None string `json:"none,omitempty"`
}

// ResourceSelector selects Kubernetes objects.
type ResourceSelector struct {
	// APIVersion of the objects, e.g. "v1" or "apps/v1".
	APIVersion string `json:"apiVersion"`
	// Kind of the objects, e.g. "Pod".
	Kind string `json:"kind"`
	// Namespace of the objects. Empty selects all namespaces (or cluster-scoped objects).
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// Name of a single object.
	// +optional
	Name string `json:"name,omitempty"`
	// LabelSelector in kubectl syntax, e.g. "app=web,tier!=db".
	// +optional
	LabelSelector string `json:"labelSelector,omitempty"`
}

// GossCheck runs a goss spec on a machine.
type GossCheck struct {
	// Spec is the path of the goss YAML file, relative to the package root.
	Spec string `json:"spec"`
}
