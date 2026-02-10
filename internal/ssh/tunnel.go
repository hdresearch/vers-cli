package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
)

// Tunnel represents an active SSH port-forwarding tunnel.
type Tunnel struct {
	LocalPort  int
	RemoteHost string
	RemotePort int

	client   *Client
	listener net.Listener
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// Forward sets up local port forwarding: it listens on localPort and forwards
// connections through the SSH tunnel to remoteHost:remotePort on the VM.
// This is equivalent to `ssh -L localPort:remoteHost:remotePort`.
//
// If localPort is 0, the OS picks an available port (accessible via Tunnel.LocalPort).
// The returned Tunnel stays open until ctx is cancelled or Close() is called.
func (c *Client) Forward(ctx context.Context, localPort int, remoteHost string, remotePort int) (*Tunnel, error) {
	sshClient, err := c.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("SSH connect: %w", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	if err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("listen on local port %d: %w", localPort, err)
	}

	// Resolve actual port (relevant when localPort=0)
	actualPort := listener.Addr().(*net.TCPAddr).Port

	tunnelCtx, cancel := context.WithCancel(ctx)

	t := &Tunnel{
		LocalPort:  actualPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
		client:     c,
		listener:   listener,
		cancel:     cancel,
	}

	// Accept loop
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		for {
			localConn, err := listener.Accept()
			if err != nil {
				select {
				case <-tunnelCtx.Done():
					return
				default:
					// Listener closed
					return
				}
			}

			remoteAddr := fmt.Sprintf("%s:%d", remoteHost, remotePort)
			remoteConn, err := sshClient.Dial("tcp", remoteAddr)
			if err != nil {
				localConn.Close()
				continue
			}

			t.wg.Add(1)
			go func() {
				defer t.wg.Done()
				relay(tunnelCtx, localConn, remoteConn)
			}()
		}
	}()

	// Shutdown goroutine: close listener + SSH when context is done
	go func() {
		<-tunnelCtx.Done()
		listener.Close()
		sshClient.Close()
	}()

	return t, nil
}

// Close stops the tunnel, closing the listener and all forwarded connections.
func (t *Tunnel) Close() {
	t.cancel()
	t.listener.Close()
	t.wg.Wait()
}

// relay bidirectionally copies data between two connections until one side
// closes or the context is cancelled.
func relay(ctx context.Context, a, b net.Conn) {
	done := make(chan struct{}, 2)

	cp := func(dst, src net.Conn) {
		io.Copy(dst, src)
		// Signal the other direction to stop. For TCP this sends a FIN.
		if tc, ok := dst.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
		done <- struct{}{}
	}

	go cp(a, b)
	go cp(b, a)

	// Wait for either both copies to finish or context cancellation
	select {
	case <-ctx.Done():
	case <-done:
		// One direction finished, wait briefly for the other
		select {
		case <-done:
		case <-ctx.Done():
		}
	}

	a.Close()
	b.Close()
}
