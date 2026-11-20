package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path"
	"time"
)

// we only need these two methods from our OSCommand struct, for killing commands
type CmdKiller interface {
	Kill(cmd *exec.Cmd) error
	PrepareForChildren(cmd *exec.Cmd)
}

type SSHHandler struct {
	oSCommand CmdKiller

	dialContext func(ctx context.Context, network, addr string) (io.Closer, error)
	startCmd    func(*exec.Cmd) error
	tempDir     func(dir string, pattern string) (name string, err error)
	getenv      func(key string) string
	setenv      func(key, value string) error
}

func NewSSHHandler(oSCommand CmdKiller) *SSHHandler {
	return &SSHHandler{
		oSCommand: oSCommand,

		dialContext: func(ctx context.Context, network, addr string) (io.Closer, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		},
		startCmd: func(cmd *exec.Cmd) error { return cmd.Start() },
		tempDir:  os.MkdirTemp,
		getenv:   os.Getenv,
		setenv:   os.Setenv,
	}
}

// HandleSSHContainerHost overrides the podman host environment variable
// to point towards a local unix socket tunneled over SSH to the specified ssh host.
func (self *SSHHandler) HandleSSHContainerHost() (io.Closer, error) {
	const key = podmanHostEnvKey
	ctx := context.Background()
	u, err := url.Parse(self.getenv(key))
	if err != nil {
		// if no or an invalid podman host is specified, continue nominally
		return noopCloser{}, nil
	}

	// if the podman host scheme is "ssh", forward the podman socket before creating the client
	if u.Scheme == "ssh" {
		tunnel, err := self.createContainerHostTunnel(ctx, u.Host, remotePodmanSocket(u))
		if err != nil {
			return noopCloser{}, fmt.Errorf("tunnel ssh podman host: %w", err)
		}
		err = self.setenv(key, tunnel.socketPath)
		if err != nil {
			return noopCloser{}, fmt.Errorf("override %s to tunneled socket: %w", key, err)
		}

		return tunnel, nil
	}
	return noopCloser{}, nil
}

type noopCloser struct{}

// podmanHostEnvKey is the environment variable that carries the Podman
// endpoint, mirroring `podman --remote`'s CONTAINER_HOST.
const podmanHostEnvKey = "CONTAINER_HOST"

// defaultRemotePodmanSocket is the remote socket we tunnel to when the SSH URL
// does not name one (e.g. `ssh://user@host`).
const defaultRemotePodmanSocket = "/run/podman/podman.sock"

// remotePodmanSocket returns the socket on the remote machine that we should
// forward. Podman endpoints look like
// `ssh://user@host/run/user/1000/podman/podman.sock`; when no path is given we
// fall back to the rootful default.
func remotePodmanSocket(u *url.URL) string {
	if u.Path != "" && u.Path != "/" {
		return u.Path
	}

	return defaultRemotePodmanSocket
}

func (noopCloser) Close() error { return nil }

type tunneledContainerHost struct {
	socketPath string
	cmd        *exec.Cmd
	socketDir  string
	oSCommand  CmdKiller
}

var _ io.Closer = (*tunneledContainerHost)(nil)

func (t *tunneledContainerHost) Close() error {
	err := t.oSCommand.Kill(t.cmd)
	_ = os.RemoveAll(t.socketDir)

	return err
}

func (self *SSHHandler) createContainerHostTunnel(ctx context.Context, remoteHost string, remoteSocket string) (*tunneledContainerHost, error) {
	socketDir, err := self.tempDir("/tmp", "lazypodman-sshtunnel-")
	if err != nil {
		return nil, fmt.Errorf("create ssh tunnel tmp file: %w", err)
	}
	localSocket := path.Join(socketDir, "podman.sock")

	cmd, err := self.tunnelSSH(ctx, remoteHost, remoteSocket, localSocket)
	if err != nil {
		return nil, fmt.Errorf("tunnel podman host over ssh: %w", err)
	}

	// set a reasonable timeout, then wait for the socket to dial successfully
	// before attempting to create a new podman client
	const socketTunnelTimeout = 8 * time.Second
	ctx, cancel := context.WithTimeout(ctx, socketTunnelTimeout)
	defer cancel()

	err = self.retrySocketDial(ctx, localSocket)
	if err != nil {
		_ = self.oSCommand.Kill(cmd)
		_ = os.RemoveAll(socketDir)

		return nil, fmt.Errorf("ssh tunneled socket never became available: %w", err)
	}

	// construct the new CONTAINER_HOST url with the proper scheme
	newContainerHostURL := url.URL{Scheme: "unix", Path: localSocket}
	return &tunneledContainerHost{
		socketPath: newContainerHostURL.String(),
		cmd:        cmd,
		socketDir:  socketDir,
		oSCommand:  self.oSCommand,
	}, nil
}

// Attempt to dial the socket until it becomes available.
// The retry loop will continue until the parent context is canceled.
func (self *SSHHandler) retrySocketDial(ctx context.Context, socketPath string) error {
	t := time.NewTicker(1 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
		// attempt to dial the socket, exit on success
		err := self.tryDial(ctx, socketPath)
		if err != nil {
			continue
		}
		return nil
	}
}

// Try to dial the specified unix socket, immediately close the connection if successfully created.
func (self *SSHHandler) tryDial(ctx context.Context, socketPath string) error {
	conn, err := self.dialContext(ctx, "unix", socketPath)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}

func (self *SSHHandler) tunnelSSH(ctx context.Context, host, remoteSocket, localSocket string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, "ssh", "-L", localSocket+":"+remoteSocket, host, "-N")
	self.oSCommand.PrepareForChildren(cmd)
	err := self.startCmd(cmd)
	if err != nil {
		return nil, err
	}
	return cmd, nil
}
