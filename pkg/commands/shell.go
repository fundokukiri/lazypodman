package commands

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"
	"time"
)

// ShellSession is a line-oriented shell inside a container, used by the Shell tab.
// It is deliberately not a PTY: no terminal emulation, no full-screen apps.
type ShellSession struct {
	stdin  io.WriteCloser
	output *io.PipeReader
	writer *io.PipeWriter

	cancel    context.CancelFunc
	closeOnce sync.Once
	closeErr  error
}

// ShellBudget bounds how long closing waits for the child to die.
var ShellBudget = 2 * time.Second

// OpenShell starts `podman exec -i -- <ref> /bin/sh` and returns a session whose
// stdin accepts one command line at a time. The caller must Close it.
func (c *cliPodmanClient) OpenShell(ctx context.Context, ref string) (*ShellSession, error) {
	if err := validatePodRef(ref); err != nil {
		return nil, err
	}

	execCtx, cancel := context.WithCancel(ctx)

	command := exec.CommandContext(execCtx, c.binary, "exec", "-i", "--", ref, "/bin/sh")
	command.Cancel = func() error { return command.Process.Kill() }
	command.WaitDelay = 2 * time.Second

	stdin, err := command.StdinPipe()
	if err != nil {
		cancel()

		return nil, err
	}

	reader, writer := io.Pipe()
	command.Stdout = writer
	command.Stderr = writer

	if err := command.Start(); err != nil {
		cancel()
		_ = stdin.Close()
		_ = writer.Close()

		return nil, err
	}

	session := &ShellSession{
		stdin:  stdin,
		output: reader,
		writer: writer,
		cancel: cancel,
	}

	// Reap the child so it does not linger as a zombie while the tab is open.
	go func() {
		_ = command.Wait()
		_ = writer.Close()
	}()

	return session, nil
}

// Output is the merged stdout/stderr stream. Reading stops when the shell exits.
func (s *ShellSession) Output() io.Reader {
	return s.output
}

// Send writes one command line to the shell.
func (s *ShellSession) Send(command string) error {
	if s == nil {
		return errors.New("no shell session")
	}

	_, err := io.WriteString(s.stdin, command+"\n")

	return err
}

// Close kills the shell and releases the pipes. Safe to call twice.
func (s *ShellSession) Close() error {
	if s == nil {
		return nil
	}

	s.closeOnce.Do(func() {
		_ = s.stdin.Close()
		s.cancel()
		_ = s.writer.Close()
		s.closeErr = s.output.Close()
	})

	return s.closeErr
}
