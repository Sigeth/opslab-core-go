// Package project holds the identifiers that will change when the project is
// renamed or moved to a GitHub organization. Keep every occurrence of the
// project name, API group, owner and target platform here.
//
// Exception: kubebuilder markers (e.g. +groupName in api/v1alpha1) cannot
// reference Go constants and must be updated by hand.
package project

const (
	// Name is the (working) project name.
	Name = "opslab"
	// APIGroup is the Kubernetes API group of all opslab resources.
	APIGroup = "opslab.io"
	// LabelPrefix prefixes every label and annotation set by opslab.
	LabelPrefix = APIGroup + "/"
	// Owner is the GitHub account hosting the repositories.
	Owner = "Sigeth"
	// ImageRegistry is where official images are published.
	ImageRegistry = "ghcr.io/sigeth"
	// OS and Arch are the only supported target platform.
	OS   = "linux"
	Arch = "amd64"
)
