// Command schemagen writes the JSON Schemas of the documents challenge authors
// and catalog publishers write by hand. Editors use them for completion and
// inline validation (e.g. a "# yaml-language-server: $schema=..." comment).
//
// Usage (from the repository root): go run ./cmd/schemagen -o schema
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"

	"github.com/invopop/jsonschema"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Sigeth/opslab-core-go/api/v1alpha1"
	"github.com/Sigeth/opslab-core-go/pkg/project"
)

const module = "github.com/Sigeth/opslab-core-go"

var schemaBaseURL = fmt.Sprintf("https://raw.githubusercontent.com/%s/opslab-core-go/main/schema/", project.Owner)

func main() {
	out := flag.String("o", "schema", "output directory")
	flag.Parse()

	docs := []struct {
		file string
		kind string
		v    any
	}{
		{"challenge-package.schema.json", v1alpha1.ChallengePackageKind, &v1alpha1.ChallengePackage{}},
		{"catalog-index.schema.json", v1alpha1.CatalogIndexKind, &v1alpha1.CatalogIndex{}},
	}
	for _, d := range docs {
		s, err := generate(d.kind, d.v)
		if err != nil {
			log.Fatalf("%s: %v", d.file, err)
		}
		s.ID = jsonschema.ID(schemaBaseURL + d.file)
		data, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(*out, d.file), append(data, '\n'), 0o644); err != nil {
			log.Fatal(err)
		}
	}
}

func generate(kind string, v any) (*jsonschema.Schema, error) {
	r := &jsonschema.Reflector{Mapper: mapType}
	if err := r.AddGoComments(module, "./api/v1alpha1"); err != nil {
		return nil, err
	}
	s := r.Reflect(v)

	// Pin apiVersion and kind so editors flag a wrong header immediately.
	root, ok := s.Definitions[kind]
	if !ok {
		return nil, fmt.Errorf("definition %s not found", kind)
	}
	for name, value := range map[string]string{"apiVersion": v1alpha1.GroupVersion.String(), "kind": kind} {
		if p, ok := root.Properties.Get(name); ok {
			p.Const = value
		}
	}
	return s, nil
}

func enum[T ~string](values []T) *jsonschema.Schema {
	s := &jsonschema.Schema{Type: "string"}
	for _, v := range values {
		s.Enum = append(s.Enum, string(v))
	}
	return s
}

// mapType gives a schema to types whose JSON form differs from their Go
// structure, and to string enums (kubebuilder markers are not read).
func mapType(t reflect.Type) *jsonschema.Schema {
	switch t {
	case reflect.TypeFor[metav1.Duration]():
		return &jsonschema.Schema{
			Type:        "string",
			Pattern:     `^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`,
			Description: `A duration such as "30m", "1h30m" or "90s".`,
		}
	case reflect.TypeFor[resource.Quantity]():
		return &jsonschema.Schema{
			AnyOf: []*jsonschema.Schema{
				{Type: "string", Pattern: `^[0-9]+(\.[0-9]+)?(m|k|Ki|M|Mi|G|Gi|T|Ti)?$`},
				{Type: "number"},
			},
			Description: `A Kubernetes quantity such as "500m", "1", "512Mi" or "2Gi".`,
		}
	case reflect.TypeFor[v1alpha1.Runtime]():
		return enum(v1alpha1.Runtimes)
	case reflect.TypeFor[v1alpha1.Category]():
		return enum(v1alpha1.Categories)
	case reflect.TypeFor[v1alpha1.Difficulty]():
		return enum(v1alpha1.Difficulties)
	case reflect.TypeFor[v1alpha1.EgressMode]():
		return enum(v1alpha1.EgressModes)
	case reflect.TypeFor[v1alpha1.ValidationMode]():
		return enum(v1alpha1.ValidationModes)
	case reflect.TypeFor[v1alpha1.SSHLoginMethod]():
		return enum([]v1alpha1.SSHLoginMethod{v1alpha1.SSHLoginKey, v1alpha1.SSHLoginPassword})
	case reflect.TypeFor[v1alpha1.AccessExpectation]():
		return enum([]v1alpha1.AccessExpectation{v1alpha1.ExpectAllowed, v1alpha1.ExpectDenied})
	case reflect.TypeFor[v1alpha1.PortState]():
		return enum([]v1alpha1.PortState{v1alpha1.PortOpen, v1alpha1.PortClosed})
	case reflect.TypeFor[v1alpha1.Protocol]():
		return enum([]v1alpha1.Protocol{v1alpha1.ProtocolTCP, v1alpha1.ProtocolUDP})
	}
	return nil
}
