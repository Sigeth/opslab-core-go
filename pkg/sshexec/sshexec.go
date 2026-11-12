// Package sshexec runs commands on vm machines over SSH, accepting only the
// host keys it is given, and generates the keys the operator injects.
package sshexec

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// MaxOutput caps the stdout and stderr kept from a command.
const MaxOutput = 1 << 20

// DefaultConnectTimeout bounds the TCP connection and SSH handshake.
const DefaultConnectTimeout = 10 * time.Second

// Request is a command to run on a machine.
type Request struct {
	// Addr is host:port.
	Addr string
	// User to log in as, with PrivateKey (OpenSSH PEM) and/or Password.
	User       string
	PrivateKey []byte
	Password   string
	// KnownHosts lists the accepted host keys (known_hosts format).
	KnownHosts []byte
	// Command run by the user's shell.
	Command string
	// Stdin is sent to the command when not nil.
	Stdin io.Reader
	// ConnectTimeout defaults to DefaultConnectTimeout.
	ConnectTimeout time.Duration
}

// Result of a command that ran.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Executor runs commands over SSH.
type Executor interface {
	// Run runs a command and waits for it. The error is set only when the
	// command could not run (unreachable, unknown host key, login refused...);
	// a command that ran and failed has a non-zero ExitCode.
	Run(ctx context.Context, req Request) (Result, error)
}

// ErrAuthFailed is returned when the server refused every credential: the
// connection and host key were fine.
var ErrAuthFailed = errors.New("ssh: login refused")

// New returns an Executor using the network.
func New() Executor { return executor{} }

type executor struct{}

func (executor) Run(ctx context.Context, req Request) (Result, error) {
	client, err := connect(ctx, req)
	if err != nil {
		return Result{}, err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return Result{}, err
	}
	defer session.Close()
	stdout, stderr := &limitedBuffer{max: MaxOutput}, &limitedBuffer{max: MaxOutput}
	session.Stdout, session.Stderr = stdout, stderr
	if req.Stdin != nil {
		session.Stdin = req.Stdin
	}

	done := make(chan error, 1)
	go func() { done <- session.Run(req.Command) }()
	select {
	case <-ctx.Done():
		client.Close()
		return Result{}, ctx.Err()
	case err = <-done:
	}
	res := Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	var exit *ssh.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exit):
		res.ExitCode = exit.ExitStatus()
	default:
		return Result{}, err
	}
	return res, nil
}

// Login reports whether the server accepts the request's credentials. The
// error is set when the login could not be attempted (unreachable server,
// unexpected host key...): a refused login is (false, nil).
func Login(ctx context.Context, req Request) (bool, error) {
	client, err := connect(ctx, req)
	if errors.Is(err, ErrAuthFailed) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	client.Close()
	return true, nil
}

// connect opens an authenticated connection.
func connect(ctx context.Context, req Request) (*ssh.Client, error) {
	var auth []ssh.AuthMethod
	if len(req.PrivateKey) > 0 {
		signer, err := ssh.ParsePrivateKey(req.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("parsing the private key: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}
	if req.Password != "" {
		answer := func(_, _ string, questions []string, _ []bool) ([]string, error) {
			answers := make([]string, len(questions))
			for i := range answers {
				answers[i] = req.Password
			}
			return answers, nil
		}
		auth = append(auth, ssh.Password(req.Password), ssh.KeyboardInteractive(answer))
	}
	if len(auth) == 0 {
		return nil, errors.New("no credentials: set a private key or a password")
	}
	host, err := hostKeys(req.KnownHosts, req.Addr)
	if err != nil {
		return nil, err
	}
	timeout := req.ConnectTimeout
	if timeout <= 0 {
		timeout = DefaultConnectTimeout
	}
	config := &ssh.ClientConfig{
		User:              req.User,
		Auth:              auth,
		HostKeyCallback:   host.check,
		HostKeyAlgorithms: host.algorithms,
		Timeout:           timeout,
	}

	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", req.Addr)
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	c, chans, reqs, err := ssh.NewClientConn(conn, req.Addr, config)
	if err != nil {
		conn.Close()
		// x/crypto/ssh has no typed error for a refused login.
		if strings.Contains(err.Error(), "unable to authenticate") {
			return nil, fmt.Errorf("%w: %v", ErrAuthFailed, err)
		}
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	return ssh.NewClient(c, chans, reqs), nil
}

// knownHost holds the accepted keys of one host.
type knownHost struct {
	keys       []ssh.PublicKey
	algorithms []string
}

func (h knownHost) check(_ string, _ net.Addr, key ssh.PublicKey) error {
	for _, k := range h.keys {
		if bytes.Equal(k.Marshal(), key.Marshal()) {
			return nil
		}
	}
	return errors.New("the host key is not the expected one")
}

// hostKeys returns the keys of KnownHosts that match the host of addr.
func hostKeys(knownHosts []byte, addr string) (knownHost, error) {
	want := knownhosts.Normalize(addr)
	var h knownHost
	rest := knownHosts
	for len(rest) > 0 {
		_, hosts, key, _, next, err := ssh.ParseKnownHosts(rest)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return h, fmt.Errorf("parsing known hosts: %w", err)
		}
		rest = next
		for _, pattern := range hosts {
			if knownhosts.Normalize(pattern) == want {
				h.keys = append(h.keys, key)
				h.algorithms = append(h.algorithms, algorithms(key)...)
			}
		}
	}
	if len(h.keys) == 0 {
		return h, fmt.Errorf("no known host key for %s", want)
	}
	return h, nil
}

// algorithms are the host key algorithms that can verify key.
func algorithms(key ssh.PublicKey) []string {
	if key.Type() == ssh.KeyAlgoRSA {
		return []string{ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256}
	}
	return []string{key.Type()}
}

// Key is a generated SSH key pair.
type Key struct {
	// Private is the private key, OpenSSH PEM.
	Private []byte
	// Public is the public key in authorized_keys format, without newline.
	Public string
}

// GenerateKey returns a new ed25519 key pair.
func GenerateKey() (Key, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return Key{}, err
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return Key{}, err
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return Key{}, err
	}
	return Key{
		Private: pem.EncodeToMemory(block),
		Public:  strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub))),
	}, nil
}

// ParseKey returns the key pair of a private key (OpenSSH PEM).
func ParseKey(private []byte) (Key, error) {
	signer, err := ssh.ParsePrivateKey(private)
	if err != nil {
		return Key{}, err
	}
	return Key{Private: private, Public: strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))}, nil
}

// KnownHostsLine is the known_hosts line accepting publicKey (authorized_keys
// format) for host (a name or address, with an optional :port).
func KnownHostsLine(host, publicKey string) (string, error) {
	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return "", err
	}
	return knownhosts.Line([]string{host}, key), nil
}

// limitedBuffer keeps the first max bytes written to it.
type limitedBuffer struct {
	bytes.Buffer
	max int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if room := b.max - b.Len(); room > 0 {
		b.Buffer.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}
