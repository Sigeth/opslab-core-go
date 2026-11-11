package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"sigs.k8s.io/yaml"
)

// The committed schema must accept the example packages accepted by
// pkg/challenge, and reject obvious mistakes.
func TestSchemaMatchesExamples(t *testing.T) {
	raw, err := os.ReadFile("../../schema/challenge-package.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("schema.json", doc); err != nil {
		t.Fatal(err)
	}
	sch, err := c.Compile("schema.json")
	if err != nil {
		t.Fatal(err)
	}

	manifests, _ := filepath.Glob("../../pkg/challenge/testdata/*/challenge.yaml")
	if len(manifests) == 0 {
		t.Fatal("no example manifests found")
	}
	for _, path := range manifests {
		t.Run(filepath.Base(filepath.Dir(path)), func(t *testing.T) {
			if err := sch.Validate(loadYAML(t, path)); err != nil {
				t.Fatalf("valid example rejected: %v", err)
			}
		})
	}

	bad := loadYAML(t, manifests[0]).(map[string]any)
	bad["spec"].(map[string]any)["machines"].([]any)[0].(map[string]any)["runtime"] = "docker"
	if err := sch.Validate(bad); err == nil {
		t.Fatal("unknown runtime accepted")
	}
}

func loadYAML(t *testing.T, path string) any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	js, err := yaml.YAMLToJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	v, err := jsonschema.UnmarshalJSON(bytes.NewReader(js))
	if err != nil {
		t.Fatal(err)
	}
	return v
}
