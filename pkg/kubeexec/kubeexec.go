// Package kubeexec runs commands in containers through the Kubernetes API
// (the operator runs setup scripts with it, the checker runs checks).
package kubeexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	utilexec "k8s.io/client-go/util/exec"
)

// MaxOutput bounds the stdout and stderr kept from a command. The rest is dropped.
const MaxOutput = 1 << 20

// Request is a command to run in a container.
type Request struct {
	Namespace string
	Pod       string
	Container string
	Command   []string
	// Stdin is sent to the command when not nil.
	Stdin io.Reader
}

// Result of a command that ran.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Executor runs commands in containers.
type Executor interface {
	// Exec runs a command and waits for it. The error is set only when the
	// command could not run (pod gone, permission denied...); a command that
	// ran and failed has a non-zero ExitCode.
	Exec(ctx context.Context, req Request) (Result, error)
}

// New returns an Executor using the API server of config.
func New(config *rest.Config) (Executor, error) {
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &executor{config: config, client: cs}, nil
}

type executor struct {
	config *rest.Config
	client kubernetes.Interface
}

func (e *executor) Exec(ctx context.Context, req Request) (Result, error) {
	url := e.client.CoreV1().RESTClient().Post().
		Resource("pods").Namespace(req.Namespace).Name(req.Pod).SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: req.Container,
			Command:   req.Command,
			Stdin:     req.Stdin != nil,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec).URL()

	ws, err := remotecommand.NewWebSocketExecutor(e.config, "GET", url.String())
	if err != nil {
		return Result{}, err
	}
	spdy, err := remotecommand.NewSPDYExecutor(e.config, "POST", url)
	if err != nil {
		return Result{}, err
	}
	exec, err := remotecommand.NewFallbackExecutor(ws, spdy, func(err error) bool {
		return httpstream.IsUpgradeFailure(err) || httpstream.IsHTTPSProxyError(err)
	})
	if err != nil {
		return Result{}, err
	}

	stdout, stderr := &limitedBuffer{}, &limitedBuffer{}
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{Stdin: req.Stdin, Stdout: stdout, Stderr: stderr})
	res := Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	var exitErr utilexec.ExitError
	switch {
	case err == nil:
		return res, nil
	case errors.As(err, &exitErr) && exitErr.Exited():
		res.ExitCode = exitErr.ExitStatus()
		return res, nil
	default:
		return res, fmt.Errorf("exec in %s/%s: %w", req.Namespace, req.Pod, err)
	}
}

// limitedBuffer keeps the first MaxOutput bytes written to it.
type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if room := MaxOutput - b.Len(); room > 0 {
		b.Buffer.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}
