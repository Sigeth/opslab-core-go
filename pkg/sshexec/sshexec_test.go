package sshexec

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

// server is an SSH server accepting one user key. Its commands: "echo"
// copies stdin to stdout, "fail" exits with 3.
type server struct {
	addr    string
	hostKey Key
}

func newServer(t *testing.T, user string, authorized Key) *server {
	t.Helper()
	hostKey := mustKey(t)
	signer, err := ssh.ParsePrivateKey(hostKey.Private)
	if err != nil {
		t.Fatal(err)
	}
	allowed, _, _, _, err := ssh.ParseAuthorizedKey([]byte(authorized.Public))
	if err != nil {
		t.Fatal(err)
	}
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if c.User() == user && bytes.Equal(key.Marshal(), allowed.Marshal()) {
				return nil, nil
			}
			return nil, io.EOF
		},
		PasswordCallback: func(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if c.User() == user && string(password) == testPassword {
				return nil, nil
			}
			return nil, io.EOF
		},
	}
	config.AddHostKey(signer)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go serve(conn, config)
		}
	}()
	return &server{addr: l.Addr().String(), hostKey: hostKey}
}

func serve(conn net.Conn, config *ssh.ServerConfig) {
	_, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		return
	}
	go ssh.DiscardRequests(reqs)
	for nc := range chans {
		ch, requests, err := nc.Accept()
		if err != nil {
			return
		}
		go func() {
			defer ch.Close()
			for req := range requests {
				if req.Type != "exec" {
					req.Reply(false, nil)
					continue
				}
				req.Reply(true, nil)
				command := string(req.Payload[4:])
				status := uint32(0)
				switch command {
				case "echo":
					io.Copy(ch, ch)
				case "fail":
					io.WriteString(ch.Stderr(), "failing")
					status = 3
				}
				payload := make([]byte, 4)
				binary.BigEndian.PutUint32(payload, status)
				ch.SendRequest("exit-status", false, payload)
				return
			}
		}()
	}
}

const testPassword = "s3cret"

func mustKey(t *testing.T) Key {
	t.Helper()
	k, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func knownHosts(t *testing.T, addr string, key Key) []byte {
	t.Helper()
	line, err := KnownHostsLine(addr, key.Public)
	if err != nil {
		t.Fatal(err)
	}
	return []byte("other.example ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl\n" + line + "\n")
}

func TestRun(t *testing.T) {
	user := mustKey(t)
	srv := newServer(t, "checker", user)
	req := Request{Addr: srv.addr, User: "checker", PrivateKey: user.Private, KnownHosts: knownHosts(t, srv.addr, srv.hostKey)}

	echo := req
	echo.Command, echo.Stdin = "echo", strings.NewReader("hello")
	res, err := New().Run(context.Background(), echo)
	if err != nil || res.ExitCode != 0 || string(res.Stdout) != "hello" {
		t.Fatalf("echo: %+v, %v", res, err)
	}

	fail := req
	fail.Command = "fail"
	res, err = New().Run(context.Background(), fail)
	if err != nil || res.ExitCode != 3 || string(res.Stderr) != "failing" {
		t.Fatalf("fail: %+v, %v", res, err)
	}
}

func TestRunRejectsUnknownHostKey(t *testing.T) {
	user := mustKey(t)
	srv := newServer(t, "checker", user)
	req := Request{Addr: srv.addr, User: "checker", PrivateKey: user.Private, KnownHosts: knownHosts(t, srv.addr, mustKey(t)), Command: "echo"}
	if _, err := New().Run(context.Background(), req); err == nil || !strings.Contains(err.Error(), "host key") {
		t.Fatalf("want a host key error, got %v", err)
	}
	req.KnownHosts = knownHosts(t, "elsewhere:22", srv.hostKey)
	if _, err := New().Run(context.Background(), req); err == nil || !strings.Contains(err.Error(), "no known host key") {
		t.Fatalf("want a missing host error, got %v", err)
	}
}

func TestRunRejectedLogin(t *testing.T) {
	srv := newServer(t, "checker", mustKey(t))
	req := Request{Addr: srv.addr, User: "checker", PrivateKey: mustKey(t).Private, KnownHosts: knownHosts(t, srv.addr, srv.hostKey), Command: "echo"}
	if _, err := New().Run(context.Background(), req); !errors.Is(err, ErrAuthFailed) {
		t.Fatalf("want ErrAuthFailed, got %v", err)
	}
}

func TestLogin(t *testing.T) {
	user := mustKey(t)
	srv := newServer(t, "student", user)
	known := knownHosts(t, srv.addr, srv.hostKey)
	for name, tc := range map[string]struct {
		req  Request
		want bool
	}{
		"key":            {Request{User: "student", PrivateKey: user.Private}, true},
		"other key":      {Request{User: "student", PrivateKey: mustKey(t).Private}, false},
		"other user":     {Request{User: "root", PrivateKey: user.Private}, false},
		"password":       {Request{User: "student", Password: testPassword}, true},
		"wrong password": {Request{User: "student", Password: "nope"}, false},
	} {
		tc.req.Addr, tc.req.KnownHosts = srv.addr, known
		got, err := Login(context.Background(), tc.req)
		if err != nil || got != tc.want {
			t.Errorf("%s: Login = %v, %v; want %v", name, got, err, tc.want)
		}
	}

	// Not a refusal: the login could not be attempted.
	if _, err := Login(context.Background(), Request{Addr: srv.addr, User: "student", PrivateKey: user.Private, KnownHosts: knownHosts(t, srv.addr, mustKey(t))}); err == nil {
		t.Error("a changed host key must be an error")
	}
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	closed := l.Addr().String()
	l.Close()
	if _, err := Login(context.Background(), Request{Addr: closed, User: "student", PrivateKey: user.Private, KnownHosts: knownHosts(t, closed, srv.hostKey)}); err == nil {
		t.Error("an unreachable server must be an error")
	}
}

func TestParseKey(t *testing.T) {
	key := mustKey(t)
	parsed, err := ParseKey(key.Private)
	if err != nil || parsed.Public != key.Public {
		t.Fatalf("ParseKey = %q, %v; want %q", parsed.Public, err, key.Public)
	}
}

func TestKnownHostsLineWithDefaultPort(t *testing.T) {
	key := mustKey(t)
	line, err := KnownHostsLine("box.inst-a.svc:22", key.Public)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(line, "box.inst-a.svc ") {
		t.Fatalf("line %q should name the host without the default port", line)
	}
	if _, err := hostKeys([]byte(line), "box.inst-a.svc:22"); err != nil {
		t.Fatal(err)
	}
}
