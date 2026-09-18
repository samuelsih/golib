package golib

import (
	"io"
	"net/http"
)

var (
	_ io.Reader         = ReaderFunc(nil)
	_ io.Writer         = WriterFunc(nil)
	_ io.Closer         = CloserFunc(nil)
	_ io.Seeker         = SeekerFunc(nil)
	_ io.ReaderAt       = ReaderAtFunc(nil)
	_ io.WriterAt       = WriterAtFunc(nil)
	_ io.ReaderFrom     = ReaderFromFunc(nil)
	_ io.WriterTo       = WriterToFunc(nil)
	_ io.ByteReader     = ByteReaderFunc(nil)
	_ io.ByteWriter     = ByteWriterFunc(nil)
	_ io.RuneReader     = RuneReaderFunc(nil)
	_ http.RoundTripper = RoundTripperFunc(nil)
)

type ReaderFunc func([]byte) (int, error)

// Read implements io.Reader interface.
func (r ReaderFunc) Read(p []byte) (int, error) {
	return r(p)
}

type WriterFunc func([]byte) (int, error)

// Write implements io.Writer interface.
func (w WriterFunc) Write(p []byte) (int, error) {
	return w(p)
}

type CloserFunc func() error

// Close implements io.Closer interface.
func (c CloserFunc) Close() error {
	return c()
}

type SeekerFunc func(offset int64, whence int) (int64, error)

// Seek implements io.Seeker interface.
func (s SeekerFunc) Seek(offset int64, whence int) (int64, error) {
	return s(offset, whence)
}

type ReaderAtFunc func(p []byte, off int64) (int, error)

// ReadAt implements io.ReaderAt interface.
func (r ReaderAtFunc) ReadAt(p []byte, off int64) (int, error) {
	return r(p, off)
}

type WriterAtFunc func(p []byte, off int64) (int, error)

// WriteAt implements io.WriterAt interface.
func (w WriterAtFunc) WriteAt(p []byte, off int64) (int, error) {
	return w(p, off)
}

type ReaderFromFunc func(r io.Reader) (int64, error)

// ReadFrom implements io.ReaderFrom interface.
func (r ReaderFromFunc) ReadFrom(reader io.Reader) (int64, error) {
	return r(reader)
}

type WriterToFunc func(w io.Writer) (int64, error)

// WriteTo implements io.WriterTo interface.
func (w WriterToFunc) WriteTo(writer io.Writer) (int64, error) {
	return w(writer)
}

type ByteReaderFunc func() (byte, error)

// ReadByte implements io.ByteReader interface.
func (b ByteReaderFunc) ReadByte() (byte, error) {
	return b()
}

type ByteWriterFunc func(c byte) error

// WriteByte implements io.ByteWriter interface.
func (b ByteWriterFunc) WriteByte(c byte) error {
	return b(c)
}

type RuneReaderFunc func() (rune, int, error)

// ReadRune implements io.RuneReader interface.
func (r RuneReaderFunc) ReadRune() (runeVal rune, size int, err error) {
	return r()
}

type RoundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip implements http.RoundTripper interface.
func (rt RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return rt(r)
}
