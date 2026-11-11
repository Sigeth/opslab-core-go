package challenge

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"testing/fstest"

	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/Sigeth/opslab-core-go/api/v1alpha1"
)

// InstructionsDir holds the student instructions, one markdown file per
// language (e.g. instructions/en.md).
const InstructionsDir = "instructions"

// DefaultInstructions is the instructions file shown when no other language matches.
const DefaultInstructions = InstructionsDir + "/en.md"

// MaxBundleSize bounds the size of the files shipped with a package, so they
// fit in a ConfigMap (1 MiB including metadata).
const MaxBundleSize = 900 << 10

// configMapSeparator replaces '/' in ConfigMap keys, which only allow
// letters, digits, '-', '_' and '.'.
const configMapSeparator = "__"

// Bundle returns the files that ship with a package, by path: the manifest,
// the instructions and every file the manifest references (setup scripts,
// seeds, extra manifests, goss specs). The solution and anything else in the
// repository stay out.
func Bundle(p *v1alpha1.ChallengePackage, files fs.FS) (map[string][]byte, error) {
	paths := []string{ManifestFile}
	if _, err := fs.Stat(files, InstructionsDir); err == nil {
		paths = append(paths, InstructionsDir)
	}
	for _, m := range p.Spec.Machines {
		paths = append(paths, m.Setup...)
		paths = append(paths, m.Seed...)
	}
	paths = append(paths, p.Spec.ExtraManifests...)
	for _, t := range p.Spec.Tasks {
		if t.Check != nil && t.Check.Goss != nil {
			paths = append(paths, t.Check.Goss.Spec)
		}
	}

	bundle := map[string][]byte{}
	size := 0
	for _, root := range paths {
		if !fs.ValidPath(root) || root == "." {
			return nil, fmt.Errorf("%s: must be a relative path inside the package", root)
		}
		err := fs.WalkDir(files, root, func(name string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			if !d.Type().IsRegular() {
				return fmt.Errorf("%s: only regular files can be shipped", name)
			}
			if _, done := bundle[name]; done {
				return nil
			}
			data, err := fs.ReadFile(files, name)
			if err != nil {
				return err
			}
			size += len(data)
			bundle[name] = data
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if size > MaxBundleSize {
		return nil, fmt.Errorf("package files total %d bytes, more than the %d allowed", size, MaxBundleSize)
	}
	return bundle, nil
}

// ConfigMapKey encodes a package path as a ConfigMap key: '/' becomes "__".
func ConfigMapKey(name string) (string, error) {
	if !fs.ValidPath(name) || strings.Contains(name, configMapSeparator) {
		return "", fmt.Errorf("%s: cannot be stored in a ConfigMap (invalid path or contains %q)", name, configMapSeparator)
	}
	key := strings.ReplaceAll(name, "/", configMapSeparator)
	if errs := validation.IsConfigMapKey(key); len(errs) > 0 {
		return "", fmt.Errorf("%s: cannot be stored in a ConfigMap: %s", name, strings.Join(errs, ", "))
	}
	return key, nil
}

// ConfigMapPath decodes a ConfigMap key into a package path.
func ConfigMapPath(key string) string {
	return strings.ReplaceAll(key, configMapSeparator, "/")
}

// ConfigMapData encodes a bundle as ConfigMap data: UTF-8 files go to data,
// the others to binaryData.
func ConfigMapData(bundle map[string][]byte) (data map[string]string, binaryData map[string][]byte, err error) {
	data = map[string]string{}
	binaryData = map[string][]byte{}
	var errs []error
	for _, name := range sortedKeys(bundle) {
		key, err := ConfigMapKey(name)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if content := bundle[name]; isText(content) {
			data[key] = string(content)
		} else {
			binaryData[key] = content
		}
	}
	return data, binaryData, errors.Join(errs...)
}

// FilesFromConfigMap rebuilds the package file system stored by ConfigMapData.
func FilesFromConfigMap(data map[string]string, binaryData map[string][]byte) fs.FS {
	files := fstest.MapFS{}
	for key, content := range data {
		files[ConfigMapPath(key)] = &fstest.MapFile{Data: []byte(content), Mode: 0o644}
	}
	for key, content := range binaryData {
		files[ConfigMapPath(key)] = &fstest.MapFile{Data: content, Mode: 0o644}
	}
	return files
}

// Instructions returns the instructions for a language, falling back to
// DefaultInstructions. ok is false when the package has none.
func Instructions(files fs.FS, lang string) (markdown string, ok bool) {
	for _, name := range []string{path.Join(InstructionsDir, lang+".md"), DefaultInstructions} {
		if data, err := fs.ReadFile(files, name); err == nil {
			return string(data), true
		}
	}
	return "", false
}

// Digest identifies the content of a package: the manifest and its files
// (without the manifest). It has the form "sha256:<hex>", like OCI digests,
// but is only comparable with other Digest results.
func Digest(p *v1alpha1.ChallengePackage, files map[string][]byte) (string, error) {
	manifest, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	write := func(b []byte) {
		_ = binary.Write(h, binary.BigEndian, uint64(len(b)))
		h.Write(b)
	}
	write(manifest)
	for _, name := range sortedKeys(files) {
		if name == ManifestFile {
			continue
		}
		write([]byte(name))
		write(files[name])
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func isText(b []byte) bool {
	return !slices.Contains(b, 0) && strings.ToValidUTF8(string(b), "�") == string(b)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
