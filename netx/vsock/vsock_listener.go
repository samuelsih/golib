package vsock

import (
	"errors"
	"fmt"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

// Listener is a vsock stream listener. It implements net.Listener.
type Listener struct {
	fd    int
	local Addr
}

var _ net.Listener = (*Listener)(nil)

// Listen creates a listener for port on the given vsock context ID.
func Listen(cid, port uint32) (Listener, error) {
	fd, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return Listener{}, fmt.Errorf("vsock: create socket: %w", err)
	}

	sa := &unix.SockaddrVM{CID: cid, Port: port}
	if err = unix.Bind(fd, sa); err != nil {
		_ = unix.Close(fd)
		return Listener{}, fmt.Errorf("bind() to %08x.%08x failed: %w", cid, port, err)
	}

	if err = unix.Listen(fd, unix.SOMAXCONN); err != nil {
		_ = unix.Close(fd)
		return Listener{}, fmt.Errorf("listen() on %08x.%08x failed: %w", cid, port, err)
	}

	local, err := socketAddr(fd)
	if err != nil {
		_ = unix.Close(fd)
		return Listener{}, err
	}

	return Listener{fd: fd, local: local}, nil
}

// Addr returns the listener's local address.
func (l Listener) Addr() net.Addr { return l.local }

// Accept waits for and returns the next connection to the listener.
func (l Listener) Accept() (net.Conn, error) {
	// The accepted fd must be nonblocking before os.NewFile wraps it so the
	// returned Client is pollable and supports deadlines and Close cancellation.
	fd, sa, err := unix.Accept4(l.fd, unix.SOCK_CLOEXEC|unix.SOCK_NONBLOCK)
	if err != nil {
		return nil, l.opError("accept", err)
	}

	remote, ok := sa.(*unix.SockaddrVM)
	if !ok {
		_ = unix.Close(fd)
		return nil, l.opError("accept", fmt.Errorf("vsock: unexpected remote address type %T", sa))
	}

	return Client{
		file:   os.NewFile(uintptr(fd), network),
		local:  l.local,
		remote: Addr{CID: remote.CID, Port: remote.Port},
	}, nil
}

// Close closes the listener.
func (l Listener) Close() error {
	return l.opError("close", unix.Close(l.fd))
}

func (l Listener) opError(op string, err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, unix.EBADF) {
		err = net.ErrClosed
	}

	return &net.OpError{
		Op:   op,
		Net:  network,
		Addr: l.local,
		Err:  err,
	}
}
