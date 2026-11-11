// Package checkerio is the contract between the operator and checkers
// (built-in or custom). A checker:
//
//   - reads the tasks to check from EnvTasks (comma-separated IDs; empty means all),
//   - reads how to reach each machine from the JSON file at EnvTargetsFile (Targets),
//   - finds the package files (goss specs...) under EnvPackageDir,
//   - prints one JSON v1alpha1.TaskResult per line on stdout, logs on stderr,
//   - exits 0 when it could check the tasks (even if some failed), non-zero on error.
package checkerio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Sigeth/opslab-core-go/api/v1alpha1"
)

// Environment variables set on checker containers.
const (
	EnvTasks       = "OPSLAB_TASKS"
	EnvTargetsFile = "OPSLAB_TARGETS_FILE"
	EnvPackageDir  = "OPSLAB_PACKAGE_DIR"
)

// Targets describes how the checker reaches the machines of an instance.
type Targets struct {
	// Machines by name.
	Machines map[string]Target `json:"machines"`
}

// Target describes how to reach one machine. Which fields are set depends on
// the runtime: SSH for vm (and sysbox when it runs sshd), exec for pod and
// sysbox, a kubeconfig for vcluster.
type Target struct {
	Runtime v1alpha1.Runtime `json:"runtime"`
	// Host is the address of the machine on the cluster network.
	Host string `json:"host,omitempty"`

	// SSH access, as the dedicated checker user.
	SSHPort           int32  `json:"sshPort,omitempty"`
	SSHUser           string `json:"sshUser,omitempty"`
	SSHPrivateKeyFile string `json:"sshPrivateKeyFile,omitempty"`
	// StudentPrivateKeyFile is the student's key, for sshLogin checks.
	StudentPrivateKeyFile string `json:"studentPrivateKeyFile,omitempty"`

	// Exec access through the Kubernetes API.
	Namespace string `json:"namespace,omitempty"`
	Pod       string `json:"pod,omitempty"`
	Container string `json:"container,omitempty"`

	// KubeconfigFile grants access to a virtual cluster.
	KubeconfigFile string `json:"kubeconfigFile,omitempty"`
}

// TasksFromEnv returns the task IDs to check; nil means all tasks.
func TasksFromEnv() []string {
	raw := strings.TrimSpace(os.Getenv(EnvTasks))
	if raw == "" {
		return nil
	}
	var ids []string
	for id := range strings.SplitSeq(raw, ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// ReadTargets reads the targets file.
func ReadTargets(path string) (*Targets, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t Targets
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("decoding targets %s: %w", path, err)
	}
	return &t, nil
}

// WriteResult prints one result as a JSON line.
func WriteResult(w io.Writer, r v1alpha1.TaskResult) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}

// ParseResults reads checker output. Every non-blank line must be a
// TaskResult; a task reported twice is an error.
func ParseResults(r io.Reader) ([]v1alpha1.TaskResult, error) {
	var results []v1alpha1.TaskResult
	seen := map[string]bool{}
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var res v1alpha1.TaskResult
		dec := json.NewDecoder(strings.NewReader(line))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&res); err != nil {
			return nil, fmt.Errorf("line %d: %w", n, err)
		}
		if res.Task == "" {
			return nil, fmt.Errorf("line %d: missing task", n)
		}
		if seen[res.Task] {
			return nil, fmt.Errorf("line %d: task %q reported twice", n, res.Task)
		}
		seen[res.Task] = true
		results = append(results, res)
	}
	return results, sc.Err()
}
