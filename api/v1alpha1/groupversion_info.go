// Package v1alpha1 contains the opslab API: the Kubernetes custom resources
// shared by the platform (api) and the operator, and the documents exchanged
// with challenge authors (challenge package manifest, catalog index) and
// checkers (task results).
//
// +kubebuilder:object:generate=true
// +groupName=opslab.io
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Sigeth/opslab-core-go/pkg/project"
)

var (
	// GroupVersion is the group and version of this API.
	GroupVersion = schema.GroupVersion{Group: project.APIGroup, Version: "v1alpha1"}

	// SchemeBuilder registers the types of this API.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types of this API to a scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(GroupVersion,
		&PlatformConfig{}, &PlatformConfigList{},
		&CatalogSource{}, &CatalogSourceList{},
		&Challenge{}, &ChallengeList{},
		&ChallengeInstance{}, &ChallengeInstanceList{},
		&CheckRun{}, &CheckRunList{},
	)
	metav1.AddToGroupVersion(s, GroupVersion)
	return nil
}
