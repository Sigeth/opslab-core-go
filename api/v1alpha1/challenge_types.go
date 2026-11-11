package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ChallengeSource is where an installed challenge comes from. Exactly one
// field must be set.
// +kubebuilder:validation:XValidation:rule="(has(self.catalog) ? 1 : 0) + (has(self.oci) ? 1 : 0) + (has(self.inline) ? 1 : 0) == 1",message="exactly one of catalog, oci and inline must be set"
type ChallengeSource struct {
	// Catalog installs a package listed by a CatalogSource.
	// +optional
	Catalog *CatalogPackageRef `json:"catalog,omitempty"`
	// OCI installs a package from an OCI reference.
	// +optional
	OCI *OCIPackageRef `json:"oci,omitempty"`
	// Inline embeds the package. For local development only: inline packages
	// are not signed.
	// +optional
	Inline *InlinePackage `json:"inline,omitempty"`
}

// CatalogPackageRef references a package of a CatalogSource.
type CatalogPackageRef struct {
	// Source is the name of the CatalogSource.
	Source string `json:"source"`
	// Name of the package.
	Name string `json:"name"`
	// Version of the package.
	Version string `json:"version"`
}

// OCIPackageRef references a package by OCI reference.
type OCIPackageRef struct {
	// Ref is the OCI reference, preferably pinned by digest.
	Ref string `json:"ref"`
	// Trust lists the signers whose package is accepted.
	Trust TrustPolicy `json:"trust"`
}

// InlinePackage embeds a package in the Challenge.
type InlinePackage struct {
	// Package is the challenge manifest.
	Package ChallengePackage `json:"package"`
	// FilesConfigMap is the name of a ConfigMap in the operator namespace
	// holding the package files (instructions, setup scripts...). Keys are
	// paths with '/' replaced by '__'.
	// +optional
	FilesConfigMap string `json:"filesConfigMap,omitempty"`
}

// ChallengeSpec selects the package to install.
type ChallengeSpec struct {
	// Source of the package.
	Source ChallengeSource `json:"source"`
	// Disabled hides the challenge from students without uninstalling it.
	// Running instances are not affected.
	// +optional
	Disabled bool `json:"disabled,omitempty"`
}

// ChallengeStatus reports the installation of a challenge.
type ChallengeStatus struct {
	// ObservedGeneration is the generation last processed by the operator.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Digest of the installed package artifact. New instances use it.
	// +optional
	Digest string `json:"digest,omitempty"`
	// Signer is the identity that signed the package, if any.
	// +optional
	Signer string `json:"signer,omitempty"`
	// Package is the resolved and validated manifest.
	// +optional
	Package *ChallengePackage `json:"package,omitempty"`
	// Mirrored is true when the package and its images are in the in-cluster registry.
	// +optional
	Mirrored bool `json:"mirrored,omitempty"`
	// Runnable is true when the cluster supports everything the challenge needs.
	// +optional
	Runnable bool `json:"runnable,omitempty"`
	// UnrunnableReason explains why the challenge cannot run on this cluster.
	// +optional
	UnrunnableReason string `json:"unrunnableReason,omitempty"`
	// Conditions of the challenge.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Challenge is a challenge package installed in the cluster.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=chal
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.status.package.metadata.version`
// +kubebuilder:printcolumn:name="Runnable",type=boolean,JSONPath=`.status.runnable`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type Challenge struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ChallengeSpec `json:"spec"`
	// +optional
	Status ChallengeStatus `json:"status,omitempty"`
}

// ChallengeList is a list of Challenge.
// +kubebuilder:object:root=true
type ChallengeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Challenge `json:"items"`
}
