package vsock

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

const (
	// VSockHypervisor is the hypervisor context ID.
	VSockHypervisor uint32 = 0
	// VSockLocal is the local loopback context ID.
	VSockLocal uint32 = 1
	// VSockCIDHost is the host context ID.
	VSockCIDHost uint32 = 2

	network = "vsock"
)

var (
	_ net.Conn = (*Client)(nil)
	_ net.Addr = (*Addr)(nil)
)

// Addr is a vsock endpoint address.
type Addr struct {
	CID  uint32
	Port uint32
}

// Network returns the address's network name, "vsock".
func (a Addr) Network() string { return network }

// String returns the address in CID.Port hexadecimal form.
func (a Addr) String() string {
	return fmt.Sprintf("%08x.%08x", a.CID, a.Port)
}

// IsZero reports whether the address is unset.
func (a Addr) IsZero() bool { return a == Addr{} }

// Client is a vsock stream connection. It implements net.Conn.
type Client struct {
	file   *os.File
	local  Addr
	remote Addr
}

// Dial connects to port on the local vsock context ID.
func Dial(port uint32) (Client, error) {
	cid, err := localCID()
	if err != nil {
		return Client{}, fmt.Errorf("vsock: get local CID: %w", err)
	}

	return DialWithCID(cid, port)
}

// DialWithCID connects to port on the given vsock context ID.
func DialWithCID(cid, port uint32) (Client, error) {
	fd, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return Client{}, fmt.Errorf("vsock: create socket: %w", err)
	}

	addr := &unix.SockaddrVM{CID: cid, Port: port}
	for {
		err = unix.Connect(fd, addr)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			_ = unix.Close(fd)
			return Client{}, fmt.Errorf("vsock: connect to %08x.%08x: %w", cid, port, err)
		}
		break
	}

	local, err := socketAddr(fd)
	if err != nil {
		_ = unix.Close(fd)
		return Client{}, err
	}

	if err := unix.SetNonblock(fd, true); err != nil {
		_ = unix.Close(fd)
		return Client{}, fmt.Errorf("vsock: set nonblocking mode: %w", err)
	}

	file := os.NewFile(uintptr(fd), network)
	return Client{
		file:   file,
		local:  local,
		remote: Addr{CID: cid, Port: port},
	}, nil
}

func localCID() (uint32, error) {
	f, err := os.Open("/dev/vsock")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	return unix.IoctlGetUint32(int(f.Fd()), unix.IOCTL_VM_SOCKETS_GET_LOCAL_CID)
}

func socketAddr(fd int) (Addr, error) {
	sa, err := unix.Getsockname(fd)
	if err != nil {
		return Addr{}, fmt.Errorf("vsock: get local address: %w", err)
	}

	addr, ok := sa.(*unix.SockaddrVM)
	if !ok {
		return Addr{}, fmt.Errorf("vsock: unexpected local address type %T", sa)
	}

	cid := addr.CID
	if cid == unix.VMADDR_CID_ANY {
		if local, err := localCID(); err == nil {
			cid = local
		}
	}

	return Addr{CID: cid, Port: addr.Port}, nil
}

// Read implements net.Conn.
func (c Client) Read(p []byte) (int, error) {
	n, err := c.file.Read(p)
	return n, c.opError("read", err)
}

// Write implements net.Conn.
func (c Client) Write(p []byte) (int, error) {
	n, err := c.file.Write(p)
	return n, c.opError("write", err)
}

// Close implements net.Conn.
func (c Client) Close() error {
	return c.opError("close", c.file.Close())
}

// LocalAddr implements net.Conn.
func (c Client) LocalAddr() net.Addr {
	if c.local.IsZero() {
		return nil
	}
	return c.local
}

// RemoteAddr implements net.Conn.
func (c Client) RemoteAddr() net.Addr {
	if c.remote.IsZero() {
		return nil
	}
	return c.remote
}

// SetDeadline implements net.Conn.
func (c Client) SetDeadline(t time.Time) error {
	return c.opError("set", c.file.SetDeadline(t))
}

// SetReadDeadline implements net.Conn.
func (c Client) SetReadDeadline(t time.Time) error {
	return c.opError("set", c.file.SetReadDeadline(t))
}

// SetWriteDeadline implements net.Conn.
func (c Client) SetWriteDeadline(t time.Time) error {
	return c.opError("set", c.file.SetWriteDeadline(t))
}

func (c Client) opError(op string, err error) error {
	if err == nil || err == io.EOF {
		return err
	}

	if pathErr, ok := errors.AsType[*os.PathError](err); ok {
		err = pathErr.Err
	}

	if errors.Is(err, os.ErrClosed) {
		err = net.ErrClosed
	}

	opErr := net.OpError{
		Op:  op,
		Net: network,
		Err: err,
	}

	if op == "set" {
		opErr.Addr = c.local
	} else {
		opErr.Source = c.local
		opErr.Addr = c.remote
	}

	return &opErr
}
