// Package challenge loads, defaults and validates challenge package manifests
// (challenge.yaml). The operator, the CLI and the checker all use it so they
// agree on what a valid package is.
package challenge

import (
	"fmt"
	"io/fs"

	"sigs.k8s.io/yaml"

	"github.com/Sigeth/opslab-core-go/api/v1alpha1"
)

// ManifestFile is the name of the manifest at the root of a package.
const ManifestFile = "challenge.yaml"

// Parse strictly decodes a manifest: unknown fields are errors. It does not
// validate or default the manifest.
func Parse(data []byte) (*v1alpha1.ChallengePackage, error) {
	var p v1alpha1.ChallengePackage
	if err := yaml.UnmarshalStrict(data, &p); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", ManifestFile, err)
	}
	return &p, nil
}

// Load reads and parses the manifest at the root of a package file system,
// e.g. os.DirFS("path/to/challenge").
func Load(files fs.FS) (*v1alpha1.ChallengePackage, error) {
	data, err := fs.ReadFile(files, ManifestFile)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}
