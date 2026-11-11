package challenge

import (
	"os"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Sigeth/opslab-core-go/api/v1alpha1"
)

func load(t *testing.T, dir string) *v1alpha1.ChallengePackage {
	t.Helper()
	p, err := Load(os.DirFS("testdata/" + dir))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExamplesAreValid(t *testing.T) {
	for _, dir := range []string{"ssh-hardening", "k8s-sa-tokens"} {
		t.Run(dir, func(t *testing.T) {
			p := load(t, dir)
			if errs := Validate(p, os.DirFS("testdata/"+dir)); len(errs) > 0 {
				t.Fatalf("unexpected errors:\n%v", errs.ToAggregate())
			}
		})
	}
}

func TestParseRejectsUnknownFields(t *testing.T) {
	_, err := Parse([]byte("apiVersion: opslab.io/v1alpha1\nkind: ChallengePackage\nspec:\n  titel: typo\n"))
	if err == nil || !strings.Contains(err.Error(), "titel") {
		t.Fatalf("expected an unknown field error, got %v", err)
	}
}

func TestValidateErrors(t *testing.T) {
	tests := []struct {
		name    string
		dir     string
		mutate  func(p *v1alpha1.ChallengePackage)
		noFiles bool
		want    string // field path of the expected error
	}{
		{name: "wrong kind", mutate: func(p *v1alpha1.ChallengePackage) { p.Kind = "Challenge" }, want: "kind"},
		{name: "bad name", mutate: func(p *v1alpha1.ChallengePackage) { p.Metadata.Name = "SSH_101" }, want: "metadata.name"},
		{name: "bad version", mutate: func(p *v1alpha1.ChallengePackage) { p.Metadata.Version = "v1" }, want: "metadata.version"},
		{name: "missing title", mutate: func(p *v1alpha1.ChallengePackage) { p.Spec.Title = " " }, want: "spec.title"},
		{name: "unknown category", mutate: func(p *v1alpha1.ChallengePackage) { p.Spec.Category = "cloud" }, want: "spec.category"},
		{name: "bad platform constraint", mutate: func(p *v1alpha1.ChallengePackage) { p.Spec.Requires.Platform = "latest!" }, want: "spec.requires.platform"},
		{
			name: "allowlist without hosts",
			mutate: func(p *v1alpha1.ChallengePackage) {
				p.Spec.Permissions.Egress = v1alpha1.EgressAllowlist
			},
			want: "spec.permissions.egressAllowlist",
		},
		{
			name: "hosts without allowlist",
			mutate: func(p *v1alpha1.ChallengePackage) {
				p.Spec.Permissions.EgressAllowlist = []string{"github.com"}
			},
			want: "spec.permissions.egressAllowlist",
		},
		{
			name: "default lifetime above max",
			mutate: func(p *v1alpha1.ChallengePackage) {
				p.Spec.Lifetime.Default = &metav1.Duration{Duration: 4 * time.Hour}
			},
			want: "spec.lifetime.default",
		},
		{name: "no machines", mutate: func(p *v1alpha1.ChallengePackage) { p.Spec.Machines = nil }, want: "spec.machines"},
		{
			name: "duplicate machine",
			mutate: func(p *v1alpha1.ChallengePackage) {
				p.Spec.Machines = append(p.Spec.Machines, p.Spec.Machines[0])
			},
			want: "spec.machines[2].name",
		},
		{name: "missing image", mutate: func(p *v1alpha1.ChallengePackage) { p.Spec.Machines[0].Image = "" }, want: "spec.machines[0].image"},
		{
			name: "kubernetes on a vm",
			mutate: func(p *v1alpha1.ChallengePackage) {
				p.Spec.Machines[0].Kubernetes = "1.34"
			},
			want: "spec.machines[0].kubernetes",
		},
		{name: "missing setup file", mutate: func(p *v1alpha1.ChallengePackage) { p.Spec.Machines[0].Setup = []string{"setup/nope.sh"} }, want: "spec.machines[0].setup[0]"},
		{
			name:    "path traversal",
			noFiles: true,
			mutate: func(p *v1alpha1.ChallengePackage) {
				p.Spec.Machines[0].Setup = []string{"../../etc/passwd"}
			},
			want: "spec.machines[0].setup[0]",
		},
		{name: "unknown terminal machine", mutate: func(p *v1alpha1.ChallengePackage) { p.Spec.Access.Terminals[0].Machine = "nope" }, want: "spec.access.terminals[0].machine"},
		{
			name: "terminal on vcluster",
			dir:  "k8s-sa-tokens",
			mutate: func(p *v1alpha1.ChallengePackage) {
				p.Spec.Access.Terminals[0].Machine = "cluster"
			},
			want: "spec.access.terminals[0].machine",
		},
		{
			name: "kubeconfigFor a non-vcluster",
			dir:  "k8s-sa-tokens",
			mutate: func(p *v1alpha1.ChallengePackage) {
				p.Spec.Machines[1].KubeconfigFor = "workstation"
			},
			want: "spec.machines[1].kubeconfigFor",
		},
		{
			name: "interval in on-demand mode",
			mutate: func(p *v1alpha1.ChallengePackage) {
				p.Spec.Validation = &v1alpha1.Validation{Interval: &metav1.Duration{Duration: time.Minute}}
			},
			want: "spec.validation.interval",
		},
		{
			name: "interval too short",
			dir:  "k8s-sa-tokens",
			mutate: func(p *v1alpha1.ChallengePackage) {
				p.Spec.Validation.Interval.Duration = time.Second
			},
			want: "spec.validation.interval",
		},
		{
			name: "duplicate task",
			mutate: func(p *v1alpha1.ChallengePackage) {
				p.Spec.Tasks[1].ID = p.Spec.Tasks[0].ID
			},
			want: "spec.tasks[1].id",
		},
		{name: "missing check", mutate: func(p *v1alpha1.ChallengePackage) { p.Spec.Tasks[0].Check = nil }, want: "spec.tasks[0].check"},
		{
			name: "two check types",
			mutate: func(p *v1alpha1.ChallengePackage) {
				p.Spec.Tasks[0].Check.Shell = &v1alpha1.ShellCheck{Run: "true"}
			},
			want: "spec.tasks[0].check",
		},
		{
			name: "sshLogin on a pod",
			mutate: func(p *v1alpha1.ChallengePackage) {
				p.Spec.Tasks[0].Check.Machine = "workstation"
			},
			want: "spec.tasks[0].check.machine",
		},
		{
			name: "password method without password",
			mutate: func(p *v1alpha1.ChallengePackage) {
				p.Spec.Tasks[1].Check.SSHLogin.Password = ""
			},
			want: "spec.tasks[1].check.sshLogin.password",
		},
		{
			name: "invalid regexp",
			mutate: func(p *v1alpha1.ChallengePackage) {
				p.Spec.Tasks[2].Check.Shell.ExpectOutput = "(unclosed"
			},
			want: "spec.tasks[2].check.shell.expectOutput",
		},
		{
			name: "invalid CEL",
			dir:  "k8s-sa-tokens",
			mutate: func(p *v1alpha1.ChallengePackage) {
				p.Spec.Tasks[0].Check.KubeCEL.All = "object.spec.(("
			},
			want: "spec.tasks[0].check.kubeCEL.all",
		},
		{
			name: "two CEL modes",
			dir:  "k8s-sa-tokens",
			mutate: func(p *v1alpha1.ChallengePackage) {
				p.Spec.Tasks[0].Check.KubeCEL.Any = "true"
			},
			want: "spec.tasks[0].check.kubeCEL",
		},
		{
			name: "kubeCEL on a pod",
			dir:  "k8s-sa-tokens",
			mutate: func(p *v1alpha1.ChallengePackage) {
				p.Spec.Tasks[0].Check.Machine = "workstation"
			},
			want: "spec.tasks[0].check.machine",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.dir
			if dir == "" {
				dir = "ssh-hardening"
			}
			p := load(t, dir)
			tt.mutate(p)
			files := os.DirFS("testdata/" + dir)
			if tt.noFiles {
				files = nil
			}
			errs := Validate(p, files)
			for _, e := range errs {
				if e.Field == tt.want {
					return
				}
			}
			t.Fatalf("expected an error on %q, got:\n%v", tt.want, errs.ToAggregate())
		})
	}
}

func TestCustomCheckerMakesChecksOptional(t *testing.T) {
	p := load(t, "ssh-hardening")
	p.Spec.Validation = &v1alpha1.Validation{CheckerImage: "ghcr.io/example/checker:1"}
	for i := range p.Spec.Tasks {
		p.Spec.Tasks[i].Check = nil
	}
	if errs := Validate(p, nil); len(errs) > 0 {
		t.Fatalf("unexpected errors:\n%v", errs.ToAggregate())
	}
}

func TestSetDefaults(t *testing.T) {
	p := load(t, "ssh-hardening")
	SetDefaults(p)
	s := p.Spec
	if s.Validation.Mode != v1alpha1.ValidationOnDemand {
		t.Errorf("validation mode = %q", s.Validation.Mode)
	}
	if got := s.Access.Terminals[1]; got.User != DefaultTerminalUser || got.Label != "server" {
		t.Errorf("terminal defaults = %+v", got)
	}
	if got := s.Tasks[0].Check.SSHLogin.Method; got != v1alpha1.SSHLoginKey {
		t.Errorf("sshLogin method = %q", got)
	}
	if got := s.Tasks[2].Check.Shell; got.User != "root" || *got.ExpectExit != 0 {
		t.Errorf("shell defaults = %+v", got)
	}

	k := load(t, "k8s-sa-tokens")
	SetDefaults(k)
	if k.Spec.Permissions.Egress != v1alpha1.EgressNone {
		t.Errorf("egress = %q", k.Spec.Permissions.Egress)
	}
	if k.Spec.Lifetime.Default.Duration != DefaultLifetime || k.Spec.Lifetime.Max.Duration != DefaultMaxLifetime {
		t.Errorf("lifetime = %+v", k.Spec.Lifetime)
	}
	if k.Spec.Validation.Interval.Duration != time.Minute {
		t.Errorf("interval overwritten: %v", k.Spec.Validation.Interval)
	}
}
