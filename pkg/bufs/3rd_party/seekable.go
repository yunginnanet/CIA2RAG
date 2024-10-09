package seekable_buffer

// ==========================================================================
// CREDIT: https://gist.github.com/larryhou/7740cf517a889ac289b962e9e325f04c
// ==========================================================================

import (
	"bytes"
	"fmt"
	"io"
)

type Buffer struct {
	b bytes.Buffer
	p int
	n int
}

func (x *Buffer) Write(p []byte) (int, error) {
	n := len(p)
	t := x.p + n
	x.grow(t)
	copy(x.Bytes()[x.p:t], p)
	if t > x.n {
		x.n = t
	}
	x.p = t
	return n, nil
}

func (x *Buffer) grow(n int) {
	if n >= x.b.Cap() {
		b := x.Bytes()
		x.b.Grow(n)
		copy(x.Bytes()[:x.n], b)
	}
}

func (x *Buffer) WriteString(s string) (int, error) {
	n := len(s)
	t := x.p + n
	x.grow(t)
	copy(x.Bytes()[x.p:t], s)
	if t > x.n {
		x.n = t
	}
	x.p = t
	return n, nil
}

func (x *Buffer) Reset() {
	x.n = 0
	x.p = 0
}

func (x *Buffer) String() string { return string(x.Bytes()) }
func (x *Buffer) Bytes() []byte  { return x.b.Bytes()[:x.n] }

func (x *Buffer) Len() int { return x.n }

func (x *Buffer) Read(p []byte) (int, error) {
	start := x.p
	if start < 0 {
		start = 0
	}
	return copy(p, x.Bytes()[x.p:]), nil
}

func (x *Buffer) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		x.p = int(offset)
	case io.SeekCurrent:
		x.p = x.p + int(offset)
	case io.SeekEnd:
		x.p = x.n + int(offset)
	default:
		return -1, fmt.Errorf("unsupported whence: %d", whence)
	}
	return int64(x.p), nil
}
