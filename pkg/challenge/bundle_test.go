package challenge

import (
	"io/fs"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func TestBundleShipsReferencedFilesOnly(t *testing.T) {
	files := os.DirFS("testdata/ssh-hardening")
	p := load(t, "ssh-hardening")
	bundle, err := Bundle(p, files)
	if err != nil {
		t.Fatal(err)
	}
	got := slices.Sorted(maps.Keys(bundle))
	want := []string{"challenge.yaml", "setup/break-ssh.sh"}
	if !slices.Equal(got, want) {
		t.Fatalf("bundle = %v, want %v", got, want)
	}
}

func TestBundleWalksDirectoriesAndInstructions(t *testing.T) {
	p := load(t, "k8s-sa-tokens")
	files := fstest.MapFS{
		"challenge.yaml":                  {Data: []byte("x")},
		"instructions/en.md":              {Data: []byte("# Hi")},
		"setup/manifests/namespace.yaml":  {Data: []byte("kind: Namespace")},
		"solution/solve.sh":               {Data: []byte("secret")},
		"README.md":                       {Data: []byte("readme")},
		"setup/manifests/nested/pod.yaml": {Data: []byte("kind: Pod")},
	}
	bundle, err := Bundle(p, files)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"instructions/en.md", "setup/manifests/nested/pod.yaml"} {
		if _, ok := bundle[name]; !ok {
			t.Errorf("%s is missing from %v", name, slices.Sorted(maps.Keys(bundle)))
		}
	}
	for _, name := range []string{"solution/solve.sh", "README.md"} {
		if _, ok := bundle[name]; ok {
			t.Errorf("%s must not ship", name)
		}
	}
}

func TestBundleSizeLimit(t *testing.T) {
	p := load(t, "ssh-hardening")
	files := fstest.MapFS{
		"challenge.yaml":     {Data: []byte("x")},
		"setup/break-ssh.sh": {Data: make([]byte, MaxBundleSize+1)},
	}
	if _, err := Bundle(p, files); err == nil || !strings.Contains(err.Error(), "allowed") {
		t.Fatalf("expected a size error, got %v", err)
	}
}

func TestConfigMapRoundTrip(t *testing.T) {
	bundle := map[string][]byte{
		"instructions/en.md": []byte("# Title\n"),
		"setup/blob.bin":     {0, 1, 2},
	}
	data, binary, err := ConfigMapData(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if data["instructions__en.md"] != "# Title\n" || len(binary["setup__blob.bin"]) != 3 {
		t.Fatalf("data=%v binary=%v", data, binary)
	}
	files := FilesFromConfigMap(data, binary)
	for name, content := range bundle {
		got, err := fs.ReadFile(files, name)
		if err != nil || string(got) != string(content) {
			t.Errorf("%s: %q, %v", name, got, err)
		}
	}
	md, ok := Instructions(files, "fr")
	if !ok || md != "# Title\n" {
		t.Fatalf("instructions fallback: %q %v", md, ok)
	}
}

func TestConfigMapKeyRejectsAmbiguousPaths(t *testing.T) {
	for _, name := range []string{"setup/a__b.sh", "setup/with space.sh", "../escape"} {
		if _, err := ConfigMapKey(name); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestDigestChangesWithContent(t *testing.T) {
	p := load(t, "ssh-hardening")
	a, _ := Digest(p, map[string][]byte{"setup/x.sh": []byte("a")})
	b, _ := Digest(p, map[string][]byte{"setup/x.sh": []byte("b")})
	c, _ := Digest(p, map[string][]byte{"setup/x.sh": []byte("a"), ManifestFile: []byte("ignored")})
	if a == b || a != c || !strings.HasPrefix(a, "sha256:") {
		t.Fatalf("a=%s b=%s c=%s", a, b, c)
	}
}
