package ssh

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// Client provides native SSH connectivity over TLS.
type Client struct {
	host    string // VM ID (becomes {vm-id}.vm.vers.sh)
	keyPath string // Path to SSH private key
}

// NewClient creates a new SSH client for the given VM.
func NewClient(host, keyPath string) *Client {
	return &Client{
		host:    host,
		keyPath: keyPath,
	}
}

// hostname returns the full hostname for TLS/SSH connection.
func (c *Client) hostname() string {
	return fmt.Sprintf("%s.vm.vers.sh", c.host)
}

// Connect establishes an SSH connection over TLS.
func (c *Client) Connect(ctx context.Context) (*ssh.Client, error) {
	hostname := c.hostname()

	// Read and parse private key
	keyData, err := os.ReadFile(c.keyPath)
	if err != nil {
		return nil, fmt.Errorf("read SSH key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("parse SSH key: %w", err)
	}

	// Dial TLS with SNI
	// InsecureSkipVerify matches previous behavior with openssl s_client
	// which didn't verify certificates. The SSH layer provides its own
	// authentication via host keys (though we also skip that check).
	dialer := &tls.Dialer{
		Config: &tls.Config{
			ServerName:         hostname,
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		},
	}
	tlsConn, err := dialer.DialContext(ctx, "tcp", hostname+":443")
	if err != nil {
		return nil, fmt.Errorf("TLS dial: %w", err)
	}

	// SSH handshake over TLS connection
	sshConfig := &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // matches current behavior
		Timeout:         30 * time.Second,
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(tlsConn, hostname, sshConfig)
	if err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("SSH handshake: %w", err)
	}

	client := ssh.NewClient(sshConn, chans, reqs)

	// Start keep-alive goroutine (matches ServerAliveInterval=10, ServerAliveCountMax=6)
	go c.keepAlive(ctx, client)

	return client, nil
}

// keepAlive sends periodic keep-alive requests to prevent connection timeout.
func (c *Client) keepAlive(ctx context.Context, client *ssh.Client) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	missedCount := 0
	maxMissed := 6

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				missedCount++
				if missedCount >= maxMissed {
					client.Close()
					return
				}
			} else {
				missedCount = 0
			}
		}
	}
}

// Interactive runs an interactive shell session with PTY support.
func (c *Client) Interactive(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) error {
	client, err := c.Connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	// Get terminal size
	width, height := 80, 24
	var fd int
	var isTerm bool
	if f, ok := stdin.(*os.File); ok {
		fd = int(f.Fd())
		if term.IsTerminal(fd) {
			isTerm = true
			if w, h, err := term.GetSize(fd); err == nil {
				width, height = w, h
			}
		}
	}

	// Request PTY
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", height, width, modes); err != nil {
		return fmt.Errorf("request PTY: %w", err)
	}

	// Wire up IO
	session.Stdin = stdin
	session.Stdout = stdout
	session.Stderr = stderr

	// Start shell
	if err := session.Shell(); err != nil {
		return fmt.Errorf("start shell: %w", err)
	}

	// Handle terminal resize if we have a real terminal
	if isTerm {
		go c.watchResize(ctx, fd, session)
	}

	// Wait for session to end or context cancellation
	done := make(chan error, 1)
	go func() {
		done <- session.Wait()
	}()

	select {
	case <-ctx.Done():
		session.Close()
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// watchResize monitors terminal size changes and updates the remote PTY.
func (c *Client) watchResize(ctx context.Context, fd int, session *ssh.Session) {
	// Use SIGWINCH on Unix systems
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, sigWinch()...)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-ctx.Done():
			return
		case <-sigCh:
			if w, h, err := term.GetSize(fd); err == nil {
				_ = session.WindowChange(h, w)
			}
		}
	}
}

// Execute runs a command on the remote host.
func (c *Client) Execute(ctx context.Context, cmd string, stdout, stderr io.Writer) error {
	client, err := c.Connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	session.Stdout = stdout
	session.Stderr = stderr

	// Run with context awareness
	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		session.Close()
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// Conn returns a raw SSH connection for use with SFTP.
// Caller is responsible for closing the returned client.
func (c *Client) Conn(ctx context.Context) (*ssh.Client, error) {
	return c.Connect(ctx)
}

// DialFunc returns a function that dials through the SSH connection.
// This can be used for port forwarding or other tunneling needs.
func (c *Client) DialFunc(ctx context.Context) (func(network, addr string) (net.Conn, error), *ssh.Client, error) {
	client, err := c.Connect(ctx)
	if err != nil {
		return nil, nil, err
	}
	return client.Dial, client, nil
}

// Session represents an active SSH session that can be used for streaming.
type Session struct {
	client  *ssh.Client
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.Reader
	mu      sync.Mutex
}

// StartSession creates a new session for streaming command execution.
func (c *Client) StartSession(ctx context.Context) (*Session, error) {
	client, err := c.Connect(ctx)
	if err != nil {
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("new session: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	return &Session{
		client:  client,
		session: session,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
	}, nil
}

// Start starts a command without waiting for it to complete.
func (s *Session) Start(cmd string) error {
	return s.session.Start(cmd)
}

// Wait waits for the command to complete.
func (s *Session) Wait() error {
	return s.session.Wait()
}

// Stdin returns the stdin writer.
func (s *Session) Stdin() io.WriteCloser {
	return s.stdin
}

// Stdout returns the stdout reader.
func (s *Session) Stdout() io.Reader {
	return s.stdout
}

// Stderr returns the stderr reader.
func (s *Session) Stderr() io.Reader {
	return s.stderr
}

// Close closes the session and underlying connection.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.session.Close()
	return s.client.Close()
}
