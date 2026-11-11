package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CatalogIndexKind is the kind of a catalog index document.
const CatalogIndexKind = "CatalogIndex"

// CatalogIndex is the document a catalog publishes (as a signed OCI
// artifact) to list its packages. It is not a Kubernetes resource.
type CatalogIndex struct {
	// APIVersion must be "opslab.io/v1alpha1".
	APIVersion string `json:"apiVersion"`
	// Kind must be "CatalogIndex".
	Kind string `json:"kind"`
	// Packages published by the catalog.
	Packages []CatalogEntry `json:"packages"`
}

// CatalogEntry describes one package of a catalog.
type CatalogEntry struct {
	// Name of the package.
	Name string `json:"name"`
	// Title of the challenge.
	Title string `json:"title"`
	// Summary of the challenge.
	// +optional
	Summary string `json:"summary,omitempty"`
	// Category of the challenge.
	Category Category `json:"category"`
	// Difficulty of the challenge.
	Difficulty Difficulty `json:"difficulty"`
	// Tags of the challenge.
	// +optional
	Tags []string `json:"tags,omitempty"`
	// Versions available, newest first.
	// +kubebuilder:validation:MinItems=1
	Versions []CatalogVersion `json:"versions"`
}

// CatalogVersion is one published version of a package.
type CatalogVersion struct {
	// Version (semver).
	Version string `json:"version"`
	// Ref is the OCI reference of the package, pinned by digest
	// (registry/repository@sha256:...).
	Ref string `json:"ref"`
}

// TrustPolicy lists the signers whose packages are accepted.
// +kubebuilder:validation:XValidation:rule="(has(self.allowUnsigned) && self.allowUnsigned) || (has(self.keys) && size(self.keys) > 0) || (has(self.keyless) && size(self.keyless) > 0)",message="trust must list keys or keyless identities, or allow unsigned packages"
type TrustPolicy struct {
	// Keys are PEM-encoded cosign public keys.
	// +optional
	Keys []string `json:"keys,omitempty"`
	// Keyless identities (Sigstore certificates), e.g. a CI workflow.
	// +optional
	Keyless []KeylessIdentity `json:"keyless,omitempty"`
	// AllowUnsigned accepts unsigned packages. For local development only.
	// +optional
	AllowUnsigned bool `json:"allowUnsigned,omitempty"`
}

// KeylessIdentity matches a Sigstore keyless signing certificate.
type KeylessIdentity struct {
	// Issuer is the OIDC issuer, e.g. "https://token.actions.githubusercontent.com".
	Issuer string `json:"issuer"`
	// SubjectRegExp matches the certificate subject, e.g.
	// "^https://github.com/Sigeth/opslab-challenge-.*/.github/workflows/release.yaml@.*$".
	SubjectRegExp string `json:"subjectRegExp"`
}

// CatalogSourceSpec defines where a catalog lives and who is trusted.
type CatalogSourceSpec struct {
	// Ref is the OCI reference of the catalog index, e.g. "ghcr.io/sigeth/opslab-catalog:latest".
	Ref string `json:"ref"`
	// Trust lists the signers whose index and packages are accepted.
	Trust TrustPolicy `json:"trust"`
	// SyncInterval between index refreshes. Defaults to 1h.
	// +optional
	SyncInterval *metav1.Duration `json:"syncInterval,omitempty"`
	// Suspend stops syncing the source.
	// +optional
	Suspend bool `json:"suspend,omitempty"`
}

// CatalogSourceStatus reports the last sync of a catalog.
type CatalogSourceStatus struct {
	// ObservedGeneration is the generation last processed by the operator.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// LastSyncTime is when the index was last fetched successfully.
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
	// IndexDigest is the digest of the last verified index.
	// +optional
	IndexDigest string `json:"indexDigest,omitempty"`
	// Packages listed by the index.
	// +optional
	Packages []CatalogEntry `json:"packages,omitempty"`
	// Conditions of the source.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// CatalogSource is a catalog of challenges the platform can install from.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Ref",type=string,JSONPath=`.spec.ref`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Last sync",type=date,JSONPath=`.status.lastSyncTime`
type CatalogSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CatalogSourceSpec `json:"spec"`
	// +optional
	Status CatalogSourceStatus `json:"status,omitempty"`
}

// CatalogSourceList is a list of CatalogSource.
// +kubebuilder:object:root=true
type CatalogSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CatalogSource `json:"items"`
}
