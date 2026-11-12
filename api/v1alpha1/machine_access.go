package v1alpha1

import "github.com/Sigeth/opslab-core-go/pkg/project"

// Access to vm machines. The operator creates both users through cloud-init
// at boot (any image meeting the base image contract can do it) and keeps
// their SSH keys, with the host key of every vm machine, in the SSHSecret of
// the instance namespace.
const (
	// SSHPort is where vm machines accept SSH.
	SSHPort int32 = 22
	// StudentUser is the student's account (UID 1000, passwordless sudo):
	// the gateway opens terminals as this user.
	StudentUser = "student"
	// CheckerUser runs checks and setup scripts (passwordless sudo).
	CheckerUser = project.Name + "-checker"

	// SSHSecret is the Secret holding the SSH material of an instance.
	SSHSecret = project.Name + "-ssh"
	// SSHSecretStudentKey and SSHSecretCheckerKey are the keys of SSHSecret
	// holding the private keys (OpenSSH PEM) of StudentUser and CheckerUser.
	SSHSecretStudentKey = "student"
	SSHSecretCheckerKey = "checker"
	// SSHSecretKnownHosts is the key of SSHSecret holding the host keys of the
	// vm machines, in known_hosts format, by machine address.
	SSHSecretKnownHosts = "known_hosts"
)
