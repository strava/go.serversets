package mmh3

import "sync"

// hash function aliases
var (
	Hash32x86  = Sum32
	Hash128x64 = Sum128

	Hash32  = Sum32 //for backward compatible
	Hash128 = Sum128
)

var (
	pool128 = sync.Pool{
		New: func() interface{} {
			return New128()
		},
	}

	pool32 = sync.Pool{
		New: func() interface{} {
			return New32()
		},
	}
)

// Sum32 computes the 32-bit hash over the data.
func Sum32(key []byte) (ret uint32) {
	h := pool32.Get().(*hash32)
	h.Write(key)
	ret = h.Sum32()
	h.Reset()
	pool32.Put(h)
	return
}

// Sum128 computes the 128-bit hash over the data.
func Sum128(key []byte) (ret []byte) {
	h := pool128.Get().(*hash128)
	h.Write(key)
	ret = h.Sum(nil)
	h.Reset()
	pool128.Put(h)
	return
}
