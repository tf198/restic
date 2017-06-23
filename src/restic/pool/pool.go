package pool

import (
	"sync"

	"github.com/restic/chunker"
)

var bufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, chunker.MinSize)
	},
}

func GetBuf() []byte {
	return bufPool.Get().([]byte)
}

func FreeBuf(data []byte) {
	bufPool.Put(data)
}
