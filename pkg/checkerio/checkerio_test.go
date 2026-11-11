package checkerio

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Sigeth/opslab-core-go/api/v1alpha1"
)

func TestRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	in := []v1alpha1.TaskResult{
		{Task: "a", Passed: true},
		{Task: "b", Passed: false, Message: "root login still allowed"},
	}
	for _, r := range in {
		if err := WriteResult(&buf, r); err != nil {
			t.Fatal(err)
		}
	}
	out, err := ParseResults(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[1] != in[1] {
		t.Fatalf("got %+v", out)
	}
}

func TestParseResultsErrors(t *testing.T) {
	for name, input := range map[string]string{
		"not json":      "checking...\n",
		"unknown field": `{"task":"a","passed":true,"score":3}`,
		"missing task":  `{"passed":true}`,
		"duplicate":     "{\"task\":\"a\",\"passed\":true}\n{\"task\":\"a\",\"passed\":false}",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseResults(strings.NewReader(input)); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestParseLogSkipsLogLines(t *testing.T) {
	log := "starting checker\n{\"task\":\"a\",\"passed\":true}\nwarning: slow machine\n{\"task\":\"b\",\"passed\":false}\n"
	out, err := ParseLog(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || !out[0].Passed || out[1].Task != "b" {
		t.Fatalf("got %+v", out)
	}
	if _, err := ParseLog(strings.NewReader(`{"task":"a","passed":"yes"}`)); err == nil {
		t.Fatal("a malformed result must be an error")
	}
}

func TestTasksFromEnv(t *testing.T) {
	t.Setenv(EnvTasks, " a, ,b ")
	if got := TasksFromEnv(); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got %q", got)
	}
	t.Setenv(EnvTasks, "")
	if got := TasksFromEnv(); got != nil {
		t.Fatalf("got %q", got)
	}
}
