package challenge

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Sigeth/opslab-core-go/api/v1alpha1"
)

// Default values applied by SetDefaults.
const (
	DefaultLifetime           = time.Hour
	DefaultMaxLifetime        = 3 * time.Hour
	DefaultValidationInterval = 30 * time.Second
	MinValidationInterval     = 10 * time.Second
	DefaultTerminalUser       = "student"
	DefaultShellUser          = "root"
	DefaultHTTPPath           = "/"
	DefaultHTTPScheme         = "http"
	DefaultHTTPStatus         = 200
)

// SetDefaults fills unset optional fields in place. Call it after Validate:
// validation reports what the author wrote, defaults are for consumers.
func SetDefaults(p *v1alpha1.ChallengePackage) {
	s := &p.Spec
	if s.Permissions == nil {
		s.Permissions = &v1alpha1.Permissions{}
	}
	if s.Permissions.Egress == "" {
		s.Permissions.Egress = v1alpha1.EgressNone
	}
	if s.Lifetime == nil {
		s.Lifetime = &v1alpha1.Lifetime{}
	}
	if s.Lifetime.Default == nil {
		s.Lifetime.Default = &metav1.Duration{Duration: DefaultLifetime}
	}
	if s.Lifetime.Max == nil {
		s.Lifetime.Max = &metav1.Duration{Duration: max(DefaultMaxLifetime, s.Lifetime.Default.Duration)}
	}
	if s.Validation == nil {
		s.Validation = &v1alpha1.Validation{}
	}
	if s.Validation.Mode == "" {
		s.Validation.Mode = v1alpha1.ValidationOnDemand
	}
	if s.Validation.Mode == v1alpha1.ValidationAuto && s.Validation.Interval == nil {
		s.Validation.Interval = &metav1.Duration{Duration: DefaultValidationInterval}
	}
	for i := range s.Access.Terminals {
		t := &s.Access.Terminals[i]
		if t.User == "" {
			t.User = DefaultTerminalUser
		}
		if t.Label == "" {
			t.Label = t.Machine
		}
	}
	for i := range s.Tasks {
		if c := s.Tasks[i].Check; c != nil {
			defaultCheck(c)
		}
	}
}

func defaultCheck(c *v1alpha1.Check) {
	switch {
	case c.Shell != nil:
		if c.Shell.User == "" {
			c.Shell.User = DefaultShellUser
		}
		if c.Shell.ExpectExit == nil {
			c.Shell.ExpectExit = new(int32(0))
		}
	case c.File != nil:
		if c.File.Exists == nil {
			c.File.Exists = new(true)
		}
	case c.SSHLogin != nil:
		if c.SSHLogin.Method == "" {
			c.SSHLogin.Method = v1alpha1.SSHLoginKey
		}
	case c.Port != nil:
		if c.Port.Protocol == "" {
			c.Port.Protocol = v1alpha1.ProtocolTCP
		}
	case c.HTTP != nil:
		if c.HTTP.Path == "" {
			c.HTTP.Path = DefaultHTTPPath
		}
		if c.HTTP.Scheme == "" {
			c.HTTP.Scheme = DefaultHTTPScheme
		}
		if c.HTTP.ExpectStatus == 0 {
			c.HTTP.ExpectStatus = DefaultHTTPStatus
		}
	}
}
