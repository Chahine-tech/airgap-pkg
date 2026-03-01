package transport

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHConfig holds the parameters needed to establish an SSH connection.
type SSHConfig struct {
	Host    string // SSH hostname or IP, e.g. "node-1"
	Port    string // SSH port, e.g. "22"
	User    string // SSH login user
	KeyPath string // path to the private key file, supports "~"
}

// Tunnel opens an SSH connection to cfg.Host and starts a local TCP listener
// on a random port. Every connection accepted on the listener is forwarded
// through SSH to targetAddr (e.g. "192.168.2.2:5000").
//
// Returns:
//   - localAddr: address of the local listener, e.g. "127.0.0.1:54321"
//   - closeFn: call this to shut down the listener and the SSH connection
//   - err: non-nil if the connection or listener could not be established
func Tunnel(cfg SSHConfig, targetAddr string) (localAddr string, closeFn func(), err error) {
	keyPath, err := expandHome(cfg.KeyPath)
	if err != nil {
		return "", nil, fmt.Errorf("expanding key path %q: %w", cfg.KeyPath, err)
	}

	authMethod, err := privateKeyAuth(keyPath)
	if err != nil {
		return "", nil, fmt.Errorf("loading private key %q: %w", keyPath, err)
	}

	hostKeyCallback, err := buildHostKeyCallback()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] could not load known_hosts, host key verification disabled: %v\n", err)
		hostKeyCallback = ssh.InsecureIgnoreHostKey() //nolint:gosec
	}

	sshCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: hostKeyCallback,
	}

	sshClient, err := ssh.Dial("tcp", cfg.Host+":"+cfg.Port, sshCfg)
	if err != nil {
		return "", nil, fmt.Errorf("ssh dial to %s:%s: %w", cfg.Host, cfg.Port, err)
	}

	// Let the OS pick a free local port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = sshClient.Close()
		return "", nil, fmt.Errorf("local listener: %w", err)
	}

	// Accept loop: each incoming connection is forwarded through the SSH tunnel.
	go func() {
		for {
			local, err := listener.Accept()
			if err != nil {
				return // listener.Close() was called — normal shutdown
			}
			go forwardConn(local, sshClient, targetAddr)
		}
	}()

	close := func() {
		_ = listener.Close()
		_ = sshClient.Close()
	}

	return listener.Addr().String(), close, nil
}

// forwardConn copies data bidirectionally between a local net.Conn and a
// remote connection dialed through sshClient to targetAddr.
func forwardConn(local net.Conn, sshClient *ssh.Client, targetAddr string) {
	defer local.Close()

	remote, err := sshClient.Dial("tcp", targetAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] ssh forward dial to %s: %v\n", targetAddr, err)
		return
	}
	defer remote.Close()

	done := make(chan struct{}, 2)

	go func() {
		_, _ = io.Copy(remote, local)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(local, remote)
		done <- struct{}{}
	}()

	<-done
}

// expandHome replaces a leading "~" with the user's home directory.
func expandHome(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, path[1:]), nil
}

// privateKeyAuth reads the PEM-encoded private key at keyPath and returns
// an ssh.AuthMethod. Supports RSA, ECDSA, and Ed25519 key types.
func privateKeyAuth(keyPath string) (ssh.AuthMethod, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading key file: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}
	return ssh.PublicKeys(signer), nil
}

// buildHostKeyCallback loads ~/.ssh/known_hosts for host key verification.
// Returns an error if the file does not exist or cannot be parsed.
func buildHostKeyCallback() (ssh.HostKeyCallback, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return knownhosts.New(filepath.Join(home, ".ssh", "known_hosts"))
}
