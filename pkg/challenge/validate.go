package challenge

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"

	"cel.dev/cel-go/cel"
	"github.com/Masterminds/semver/v3"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/Sigeth/opslab-core-go/api/v1alpha1"
)

// Validate checks a manifest as written by the author (before SetDefaults).
// When files is not nil, paths referenced by the manifest must exist in it.
func Validate(p *v1alpha1.ChallengePackage, files fs.FS) field.ErrorList {
	v := validator{files: files, machines: map[string]*v1alpha1.Machine{}}
	v.header(p)
	v.spec(&p.Spec, field.NewPath("spec"))
	return v.errs
}

type validator struct {
	files    fs.FS
	machines map[string]*v1alpha1.Machine
	errs     field.ErrorList
}

func (v *validator) add(err *field.Error) { v.errs = append(v.errs, err) }

func (v *validator) header(p *v1alpha1.ChallengePackage) {
	if want := v1alpha1.GroupVersion.String(); p.APIVersion != want {
		v.add(field.NotSupported(field.NewPath("apiVersion"), p.APIVersion, []string{want}))
	}
	if p.Kind != v1alpha1.ChallengePackageKind {
		v.add(field.NotSupported(field.NewPath("kind"), p.Kind, []string{v1alpha1.ChallengePackageKind}))
	}
	meta := field.NewPath("metadata")
	v.dnsLabel(meta.Child("name"), p.Metadata.Name)
	if _, err := semver.StrictNewVersion(p.Metadata.Version); err != nil {
		v.add(field.Invalid(meta.Child("version"), p.Metadata.Version, "must be a semantic version such as 1.2.0"))
	}
}

func (v *validator) spec(s *v1alpha1.ChallengePackageSpec, fp *field.Path) {
	v.required(fp.Child("title"), s.Title)
	v.required(fp.Child("summary"), s.Summary)
	oneOf(v, fp.Child("category"), s.Category, v1alpha1.Categories)
	oneOf(v, fp.Child("difficulty"), s.Difficulty, v1alpha1.Difficulties)
	if s.EstimatedTime != nil && s.EstimatedTime.Duration <= 0 {
		v.add(field.Invalid(fp.Child("estimatedTime"), s.EstimatedTime.Duration.String(), "must be positive"))
	}
	for i, a := range s.Authors {
		v.required(fp.Child("authors").Index(i).Child("name"), a.Name)
	}
	if s.Requires != nil && s.Requires.Platform != "" {
		if _, err := semver.NewConstraint(s.Requires.Platform); err != nil {
			v.add(field.Invalid(fp.Child("requires", "platform"), s.Requires.Platform, "must be a semantic version constraint such as >=0.3.0"))
		}
	}
	v.permissions(s.Permissions, fp.Child("permissions"))
	v.lifetime(s.Lifetime, fp.Child("lifetime"))

	// Machines first: everything else references them.
	v.machineList(s.Machines, fp.Child("machines"))
	for i, m := range s.ExtraManifests {
		v.file(fp.Child("extraManifests").Index(i), m)
	}
	v.access(&s.Access, fp.Child("access"))
	customChecker := v.validation(s.Validation, fp.Child("validation"))
	v.tasks(s.Tasks, fp.Child("tasks"), customChecker)
}

func (v *validator) permissions(p *v1alpha1.Permissions, fp *field.Path) {
	if p == nil {
		return
	}
	if p.Egress != "" {
		oneOf(v, fp.Child("egress"), p.Egress, v1alpha1.EgressModes)
	}
	allowlist := fp.Child("egressAllowlist")
	switch {
	case p.Egress == v1alpha1.EgressAllowlist && len(p.EgressAllowlist) == 0:
		v.add(field.Required(allowlist, "required when egress is allowlist"))
	case p.Egress != v1alpha1.EgressAllowlist && len(p.EgressAllowlist) > 0:
		v.add(field.Forbidden(allowlist, "only allowed when egress is allowlist"))
	}
	for i, host := range p.EgressAllowlist {
		for _, msg := range validation.IsDNS1123Subdomain(strings.TrimPrefix(host, "*.")) {
			v.add(field.Invalid(allowlist.Index(i), host, msg))
		}
	}
}

func (v *validator) lifetime(l *v1alpha1.Lifetime, fp *field.Path) {
	if l == nil {
		return
	}
	if l.Default != nil && l.Default.Duration <= 0 {
		v.add(field.Invalid(fp.Child("default"), l.Default.Duration.String(), "must be positive"))
	}
	if l.Max != nil && l.Max.Duration <= 0 {
		v.add(field.Invalid(fp.Child("max"), l.Max.Duration.String(), "must be positive"))
	}
	if l.Default != nil && l.Max != nil && l.Default.Duration > l.Max.Duration {
		v.add(field.Invalid(fp.Child("default"), l.Default.Duration.String(), "must not exceed max"))
	}
}

func (v *validator) machineList(machines []v1alpha1.Machine, fp *field.Path) {
	if len(machines) == 0 {
		v.add(field.Required(fp, "at least one machine is required"))
	}
	v.maxItems(fp, len(machines), MaxMachines)
	for i := range machines {
		m := &machines[i]
		mp := fp.Index(i)
		v.dnsLabel(mp.Child("name"), m.Name)
		if _, dup := v.machines[m.Name]; dup {
			v.add(field.Duplicate(mp.Child("name"), m.Name))
		}
		v.machines[m.Name] = m
		oneOf(v, mp.Child("runtime"), m.Runtime, v1alpha1.Runtimes)

		isVCluster := m.Runtime == v1alpha1.RuntimeVCluster
		switch {
		case isVCluster && m.Image != "":
			v.add(field.Forbidden(mp.Child("image"), "not used by vcluster machines"))
		case !isVCluster && m.Image == "":
			v.add(field.Required(mp.Child("image"), ""))
		}
		if !isVCluster {
			if m.Kubernetes != "" {
				v.add(field.Forbidden(mp.Child("kubernetes"), "only allowed for vcluster machines"))
			}
			if len(m.Seed) > 0 {
				v.add(field.Forbidden(mp.Child("seed"), "only allowed for vcluster machines"))
			}
		}
		if isVCluster && len(m.Setup) > 0 {
			v.add(field.Forbidden(mp.Child("setup"), "not supported by vcluster machines: use seed"))
		}
		if r := m.Resources; r != nil {
			if r.CPU.Sign() <= 0 {
				v.add(field.Invalid(mp.Child("resources", "cpu"), r.CPU.String(), "must be positive"))
			}
			if r.Memory.Sign() <= 0 {
				v.add(field.Invalid(mp.Child("resources", "memory"), r.Memory.String(), "must be positive"))
			}
		}
		for j, s := range m.Setup {
			v.file(mp.Child("setup").Index(j), s)
		}
		for j, s := range m.Seed {
			v.file(mp.Child("seed").Index(j), s)
		}
	}
	// Second pass: references between machines.
	for i := range machines {
		m := &machines[i]
		if m.KubeconfigFor == "" {
			continue
		}
		kp := fp.Index(i).Child("kubeconfigFor")
		switch target, ok := v.machines[m.KubeconfigFor]; {
		case !ok:
			v.add(field.NotFound(kp, m.KubeconfigFor))
		case target.Runtime != v1alpha1.RuntimeVCluster:
			v.add(field.Invalid(kp, m.KubeconfigFor, "must reference a vcluster machine"))
		case m.Runtime == v1alpha1.RuntimeVCluster:
			v.add(field.Forbidden(kp, "not allowed on vcluster machines"))
		}
	}
}

func (v *validator) access(a *v1alpha1.Access, fp *field.Path) {
	if len(a.Terminals) == 0 {
		v.add(field.Required(fp.Child("terminals"), "at least one terminal is required"))
	}
	v.maxItems(fp.Child("terminals"), len(a.Terminals), MaxTerminals)
	v.maxItems(fp.Child("web"), len(a.Web), MaxWebEndpoints)
	for i, t := range a.Terminals {
		v.shellMachine(fp.Child("terminals").Index(i).Child("machine"), t.Machine)
	}
	for i, w := range a.Web {
		wp := fp.Child("web").Index(i)
		v.machineRef(wp.Child("machine"), w.Machine)
		v.port(wp.Child("port"), w.Port)
	}
	if a.IDE != nil {
		v.shellMachine(fp.Child("ide", "machine"), a.IDE.Machine)
	}
}

// validation reports whether the package uses a custom checker.
func (v *validator) validation(val *v1alpha1.Validation, fp *field.Path) bool {
	if val == nil {
		return false
	}
	if val.Mode != "" {
		oneOf(v, fp.Child("mode"), val.Mode, v1alpha1.ValidationModes)
	}
	if val.Interval != nil {
		switch {
		case val.Mode != v1alpha1.ValidationAuto:
			v.add(field.Forbidden(fp.Child("interval"), "only allowed when mode is auto"))
		case val.Interval.Duration < MinValidationInterval:
			v.add(field.Invalid(fp.Child("interval"), val.Interval.Duration.String(), "must be at least "+MinValidationInterval.String()))
		}
	}
	return val.CheckerImage != ""
}

func (v *validator) tasks(tasks []v1alpha1.Task, fp *field.Path, customChecker bool) {
	if len(tasks) == 0 {
		v.add(field.Required(fp, "at least one task is required"))
	}
	v.maxItems(fp, len(tasks), MaxTasks)
	ids := map[string]bool{}
	for i := range tasks {
		t := &tasks[i]
		tp := fp.Index(i)
		v.dnsLabel(tp.Child("id"), t.ID)
		if ids[t.ID] {
			v.add(field.Duplicate(tp.Child("id"), t.ID))
		}
		ids[t.ID] = true
		v.required(tp.Child("title"), t.Title)
		if t.Points < 0 {
			v.add(field.Invalid(tp.Child("points"), t.Points, "must not be negative"))
		}
		v.maxItems(tp.Child("hints"), len(t.Hints), MaxHints)
		for j, h := range t.Hints {
			hp := tp.Child("hints").Index(j)
			v.required(hp.Child("text"), h.Text)
			if h.Cost < 0 {
				v.add(field.Invalid(hp.Child("cost"), h.Cost, "must not be negative"))
			}
		}
		switch {
		case t.Check != nil:
			v.check(t.Check, tp.Child("check"))
		case !customChecker:
			v.add(field.Required(tp.Child("check"), "required unless validation.checkerImage is set"))
		}
	}
}

var (
	octalMode = regexp.MustCompile(`^0?[0-7]{3,4}$`)
	celEnv    = mustCELEnv()
)

func mustCELEnv() *cel.Env {
	env, err := cel.NewEnv(cel.Variable("object", cel.DynType))
	if err != nil {
		panic(err)
	}
	return env
}

func (v *validator) check(c *v1alpha1.Check, fp *field.Path) {
	set := 0
	for _, isSet := range []bool{c.Shell != nil, c.File != nil, c.SSHLogin != nil, c.Port != nil, c.HTTP != nil, c.KubeCEL != nil, c.Goss != nil} {
		if isSet {
			set++
		}
	}
	if set != 1 {
		v.add(field.Invalid(fp, set, "exactly one check type must be set (shell, file, sshLogin, port, http, kubeCEL, goss)"))
		return
	}
	mp := fp.Child("machine")
	m, ok := v.machines[c.Machine]
	if !ok {
		v.add(field.NotFound(mp, c.Machine))
	}

	switch {
	case c.Shell != nil:
		sp := fp.Child("shell")
		v.required(sp.Child("run"), c.Shell.Run)
		v.regexp(sp.Child("expectOutput"), c.Shell.ExpectOutput)
		v.needsShell(mp, m)
	case c.File != nil:
		sp := fp.Child("file")
		if !path.IsAbs(c.File.Path) {
			v.add(field.Invalid(sp.Child("path"), c.File.Path, "must be absolute"))
		}
		if c.File.Mode != "" && !octalMode.MatchString(c.File.Mode) {
			v.add(field.Invalid(sp.Child("mode"), c.File.Mode, "must be an octal mode such as 0600"))
		}
		v.regexp(sp.Child("contains"), c.File.Contains)
		v.regexp(sp.Child("notContains"), c.File.NotContains)
		v.needsShell(mp, m)
	case c.SSHLogin != nil:
		sp := fp.Child("sshLogin")
		v.required(sp.Child("user"), c.SSHLogin.User)
		if c.SSHLogin.Method != "" {
			oneOf(v, sp.Child("method"), c.SSHLogin.Method, []v1alpha1.SSHLoginMethod{v1alpha1.SSHLoginKey, v1alpha1.SSHLoginPassword})
		}
		if c.SSHLogin.Method == v1alpha1.SSHLoginPassword && c.SSHLogin.Password == "" {
			v.add(field.Required(sp.Child("password"), "required with the password method"))
		}
		oneOf(v, sp.Child("expect"), c.SSHLogin.Expect, []v1alpha1.AccessExpectation{v1alpha1.ExpectAllowed, v1alpha1.ExpectDenied})
		if m != nil && m.Runtime != v1alpha1.RuntimeVM && m.Runtime != v1alpha1.RuntimeSysbox {
			v.add(field.Invalid(mp, c.Machine, "sshLogin checks need a vm or sysbox machine"))
		}
	case c.Port != nil:
		sp := fp.Child("port")
		v.port(sp.Child("port"), c.Port.Port)
		if c.Port.Protocol != "" {
			oneOf(v, sp.Child("protocol"), c.Port.Protocol, []v1alpha1.Protocol{v1alpha1.ProtocolTCP, v1alpha1.ProtocolUDP})
		}
		oneOf(v, sp.Child("expect"), c.Port.Expect, []v1alpha1.PortState{v1alpha1.PortOpen, v1alpha1.PortClosed})
		v.needsNetwork(mp, m)
	case c.HTTP != nil:
		sp := fp.Child("http")
		v.port(sp.Child("port"), c.HTTP.Port)
		if c.HTTP.Path != "" && !strings.HasPrefix(c.HTTP.Path, "/") {
			v.add(field.Invalid(sp.Child("path"), c.HTTP.Path, "must start with /"))
		}
		if c.HTTP.Scheme != "" {
			oneOf(v, sp.Child("scheme"), c.HTTP.Scheme, []string{"http", "https"})
		}
		if s := c.HTTP.ExpectStatus; s != 0 && (s < 100 || s > 599) {
			v.add(field.Invalid(sp.Child("expectStatus"), s, "must be an HTTP status code"))
		}
		v.regexp(sp.Child("expectBody"), c.HTTP.ExpectBody)
		v.needsNetwork(mp, m)
	case c.KubeCEL != nil:
		v.kubeCEL(c.KubeCEL, fp.Child("kubeCEL"))
		if m != nil && m.Runtime != v1alpha1.RuntimeVCluster {
			v.add(field.Invalid(mp, c.Machine, "kubeCEL checks need a vcluster machine"))
		}
	case c.Goss != nil:
		v.file(fp.Child("goss", "spec"), c.Goss.Spec)
		v.needsShell(mp, m)
	}
}

func (v *validator) kubeCEL(k *v1alpha1.KubeCELCheck, fp *field.Path) {
	rp := fp.Child("resource")
	v.required(rp.Child("apiVersion"), k.Resource.APIVersion)
	v.required(rp.Child("kind"), k.Resource.Kind)
	if k.Resource.LabelSelector != "" {
		if _, err := labels.Parse(k.Resource.LabelSelector); err != nil {
			v.add(field.Invalid(rp.Child("labelSelector"), k.Resource.LabelSelector, err.Error()))
		}
	}
	exprs := map[string]string{"all": k.All, "any": k.Any, "none": k.None}
	set := 0
	for _, name := range []string{"all", "any", "none"} {
		expr := exprs[name]
		if expr == "" {
			continue
		}
		set++
		ast, issues := celEnv.Compile(expr)
		if issues != nil && issues.Err() != nil {
			v.add(field.Invalid(fp.Child(name), expr, issues.Err().Error()))
			continue
		}
		if t := ast.OutputType(); t != cel.BoolType && t != cel.DynType {
			v.add(field.Invalid(fp.Child(name), expr, "must evaluate to a bool"))
		}
	}
	if set != 1 {
		v.add(field.Invalid(fp, set, "exactly one of all, any and none must be set"))
	}
}

// --- helpers ---------------------------------------------------------------

// List limits, mirrored by the MaxItems markers in api/v1alpha1.
const (
	MaxMachines     = 16
	MaxTasks        = 64
	MaxTerminals    = 16
	MaxWebEndpoints = 16
	MaxHints        = 8
)

func (v *validator) maxItems(fp *field.Path, n, limit int) {
	if n > limit {
		v.add(field.TooMany(fp, n, limit))
	}
}

func (v *validator) required(fp *field.Path, value string) {
	if strings.TrimSpace(value) == "" {
		v.add(field.Required(fp, ""))
	}
}

func (v *validator) dnsLabel(fp *field.Path, value string) {
	if value == "" {
		v.add(field.Required(fp, ""))
		return
	}
	for _, msg := range validation.IsDNS1123Label(value) {
		v.add(field.Invalid(fp, value, msg))
	}
}

func (v *validator) port(fp *field.Path, port int32) {
	if port < 1 || port > 65535 {
		v.add(field.Invalid(fp, port, "must be between 1 and 65535"))
	}
}

func (v *validator) regexp(fp *field.Path, expr string) {
	if expr == "" {
		return
	}
	if _, err := regexp.Compile(expr); err != nil {
		v.add(field.Invalid(fp, expr, err.Error()))
	}
}

// file checks that a package-relative path is safe and, when files are
// available, exists.
func (v *validator) file(fp *field.Path, name string) {
	if !fs.ValidPath(name) || name == "." {
		v.add(field.Invalid(fp, name, "must be a relative path inside the package, without '..'"))
		return
	}
	if v.files == nil {
		return
	}
	if _, err := fs.Stat(v.files, name); err != nil {
		v.add(field.NotFound(fp, name))
	}
}

func (v *validator) machineRef(fp *field.Path, name string) *v1alpha1.Machine {
	m, ok := v.machines[name]
	if !ok {
		v.add(field.NotFound(fp, name))
		return nil
	}
	return m
}

// shellMachine checks a reference to a machine that must provide a shell.
func (v *validator) shellMachine(fp *field.Path, name string) {
	v.needsShell(fp, v.machineRef(fp, name))
}

func (v *validator) needsShell(fp *field.Path, m *v1alpha1.Machine) {
	if m != nil && m.Runtime == v1alpha1.RuntimeVCluster {
		v.add(field.Invalid(fp, m.Name, "vcluster machines have no shell: target a machine with kubeconfigFor instead"))
	}
}

func (v *validator) needsNetwork(fp *field.Path, m *v1alpha1.Machine) {
	if m != nil && m.Runtime == v1alpha1.RuntimeVCluster {
		v.add(field.Invalid(fp, m.Name, "vcluster machines cannot be probed over the network"))
	}
}

func oneOf[T ~string](v *validator, fp *field.Path, value T, allowed []T) {
	if slices.Contains(allowed, value) {
		return
	}
	if value == "" {
		v.add(field.Required(fp, fmt.Sprintf("must be one of %v", allowed)))
		return
	}
	names := make([]string, len(allowed))
	for i, a := range allowed {
		names[i] = string(a)
	}
	v.add(field.NotSupported(fp, value, names))
}
