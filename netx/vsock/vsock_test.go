package vsock

import (
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/samuelsih/golib/assert"
	"golang.org/x/sys/unix"
)

func TestAddr(t *testing.T) {
	tests := []struct {
		name string
		addr Addr
		want string
	}{
		{
			name: "hypervisor",
			addr: Addr{CID: VSockHypervisor, Port: 0},
			want: "00000000.00000000",
		},
		{
			name: "local",
			addr: Addr{CID: VSockLocal, Port: 1234},
			want: "00000001.000004d2",
		},
		{
			name: "host",
			addr: Addr{CID: VSockCIDHost, Port: 0xffffffff},
			want: "00000002.ffffffff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.addr.Network(), network)
			assert.Equal(t, tt.addr.String(), tt.want)
		})
	}
}

func TestAddrIsZero(t *testing.T) {
	assert.True(t, (Addr{}).IsZero())
	assert.False(t, (Addr{CID: VSockLocal}).IsZero())
	assert.False(t, (Addr{Port: 1}).IsZero())
}

func TestClientZeroAddr(t *testing.T) {
	var client Client

	assert.Nil(t, client.LocalAddr())
	assert.Nil(t, client.RemoteAddr())
}

func TestClientReadWrite(t *testing.T) {
	client, server, port := newVSockPair(t)

	local, ok := client.LocalAddr().(Addr)
	assert.True(t, ok)
	assert.False(t, local.IsZero())
	assert.Equal(t, local.Network(), network)

	remote, ok := client.RemoteAddr().(Addr)
	assert.True(t, ok)
	assert.False(t, remote.IsZero())
	assert.Equal(t, remote.Network(), network)
	assert.Equal(t, remote.CID, VSockLocal)
	assert.Equal(t, remote.Port, port)

	payload := []byte("hello")
	n, err := client.Write(payload)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	assert.Equal(t, n, len(payload))

	got := make([]byte, len(payload))
	readFullFD(t, server, got)
	assert.Equal(t, got, payload)

	reply := []byte("world")
	writeFullFD(t, server, reply)

	got = make([]byte, len(reply))
	n, err = io.ReadFull(client, got)
	assert.NoError(t, err)
	assert.Equal(t, n, len(reply))
	assert.Equal(t, got, reply)
}

func TestClientDeadline(t *testing.T) {
	client, server, _ := newVSockPair(t)

	if err := client.SetDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}

	n, err := client.Read(make([]byte, 1))
	assert.Equal(t, n, 0)
	assert.ErrorIs(t, err, os.ErrDeadlineExceeded)

	netErr := assert.ErrorAsType[net.Error](t, err)
	if netErr != nil {
		assert.True(t, netErr.Timeout())
	}

	assert.NoError(t, client.SetDeadline(time.Time{}))
	assert.NoError(t, client.SetReadDeadline(time.Time{}))
	assert.NoError(t, client.SetWriteDeadline(time.Time{}))

	reply := []byte("ok")
	writeFullFD(t, server, reply)

	got := make([]byte, len(reply))
	n, err = io.ReadFull(client, got)
	assert.NoError(t, err)
	assert.Equal(t, n, len(reply))
	assert.Equal(t, got, reply)
}

func TestClientClose(t *testing.T) {
	client, _, _ := newVSockPair(t)

	if err := client.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	assert.ErrorIs(t, client.Close(), net.ErrClosed)

	n, err := client.Read(make([]byte, 1))
	assert.Equal(t, n, 0)
	assert.ErrorIs(t, err, net.ErrClosed)

	n, err = client.Write([]byte("x"))
	assert.Equal(t, n, 0)
	assert.ErrorIs(t, err, net.ErrClosed)
}

func TestListenerAddr(t *testing.T) {
	listener := newVSockListener(t, VSockLocal)

	addr, ok := listener.Addr().(Addr)
	assert.True(t, ok)
	assert.False(t, addr.IsZero())
	assert.Equal(t, addr.Network(), network)
	assert.Equal(t, addr.CID, VSockLocal)
	if addr.Port == 0 || addr.Port == unix.VMADDR_PORT_ANY {
		t.Fatalf("unexpected port: %d", addr.Port)
	}
}

func TestListenerAccept(t *testing.T) {
	listener := newVSockListener(t, VSockLocal)
	local := listener.Addr().(Addr)

	accepted := make(chan connResult, 1)
	go func() {
		conn, err := listener.Accept()
		accepted <- connResult{conn: conn, err: err}
	}()

	client, err := DialWithCID(VSockLocal, local.Port)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	result := <-accepted
	if result.err != nil {
		t.Fatalf("accept: %v", result.err)
	}
	t.Cleanup(func() {
		_ = result.conn.Close()
	})

	server := result.conn

	remote, ok := server.RemoteAddr().(Addr)
	assert.True(t, ok)
	assert.False(t, remote.IsZero())
	assert.Equal(t, remote.Network(), network)
	assert.Equal(t, remote.CID, VSockLocal)

	payload := []byte("hello")
	n, err := client.Write(payload)
	assert.NoError(t, err)
	assert.Equal(t, n, len(payload))

	got := make([]byte, len(payload))
	n, err = io.ReadFull(server, got)
	assert.NoError(t, err)
	assert.Equal(t, n, len(payload))
	assert.Equal(t, got, payload)

	reply := []byte("world")
	n, err = server.Write(reply)
	assert.NoError(t, err)
	assert.Equal(t, n, len(reply))

	got = make([]byte, len(reply))
	n, err = io.ReadFull(client, got)
	assert.NoError(t, err)
	assert.Equal(t, n, len(reply))
	assert.Equal(t, got, reply)
}

func TestListenerClose(t *testing.T) {
	listener := newVSockListener(t, VSockLocal)

	assert.NoError(t, listener.Close())
	assert.ErrorIs(t, listener.Close(), net.ErrClosed)

	conn, err := listener.Accept()
	assert.Nil(t, conn)
	assert.ErrorIs(t, err, net.ErrClosed)
}

type connResult struct {
	conn net.Conn
	err  error
}

func newVSockListener(t *testing.T, cid uint32) Listener {
	t.Helper()

	listener, err := Listen(cid, unix.VMADDR_PORT_ANY)
	if err != nil {
		t.Skipf("vsock listen is unavailable: %v", err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})

	return listener
}

type acceptResult struct {
	fd  int
	err error
}

func newVSockPair(t *testing.T) (Client, int, uint32) {
	t.Helper()

	listener, port := listenVSock(t, unix.VMADDR_CID_LOCAL)
	accepted := startAccept(listener)

	client, err := DialWithCID(VSockLocal, port)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	return client, waitAccept(t, accepted), port
}

func listenVSock(t *testing.T, cid uint32) (int, uint32) {
	t.Helper()

	fd, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Skipf("vsock is unavailable: %v", err)
	}
	t.Cleanup(func() {
		_ = unix.Close(fd)
	})

	err = unix.Bind(fd, &unix.SockaddrVM{CID: cid, Port: unix.VMADDR_PORT_ANY})
	if err != nil {
		t.Skipf("vsock bind is unavailable: %v", err)
	}
	if err := unix.Listen(fd, 1); err != nil {
		t.Skipf("vsock listen is unavailable: %v", err)
	}

	sa, err := unix.Getsockname(fd)
	if err != nil {
		t.Fatalf("getsockname: %v", err)
	}

	addr, ok := sa.(*unix.SockaddrVM)
	if !ok {
		t.Fatalf("getsockname: got %T, want *unix.SockaddrVM", sa)
	}

	return fd, addr.Port
}

func startAccept(listener int) <-chan acceptResult {
	results := make(chan acceptResult, 1)
	go func() {
		fd, _, err := unix.Accept(listener)
		results <- acceptResult{fd: fd, err: err}
	}()

	return results
}

func waitAccept(t *testing.T, results <-chan acceptResult) int {
	t.Helper()

	result := <-results
	if result.err != nil {
		t.Fatal(result.err)
	}
	t.Cleanup(func() {
		_ = unix.Close(result.fd)
	})

	return result.fd
}

func readFullFD(t *testing.T, fd int, buf []byte) {
	t.Helper()

	for len(buf) > 0 {
		n, err := unix.Read(fd, buf)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if n == 0 {
			t.Fatal("vsock: unexpected EOF")
		}
		buf = buf[n:]
	}
}

func writeFullFD(t *testing.T, fd int, buf []byte) {
	t.Helper()

	for len(buf) > 0 {
		n, err := unix.Write(fd, buf)
		if err != nil {
			t.Fatalf("write: %v", err)
		}
		if n == 0 {
			t.Fatal("vsock: short write")
		}
		buf = buf[n:]
	}
}
