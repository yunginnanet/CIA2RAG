package bufs

import (
	"bytes"
	"sync"
)

var Buffers = &sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(nil)
	},
}

func GetBuffer() *bytes.Buffer {
	buf := Buffers.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func PutBuffer(buf *bytes.Buffer) {
	Buffers.Put(buf)
}
