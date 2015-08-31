package mmh3

import "sync"

var (
	Hash32x86  = Sum32
	Hash128x64 = Sum128

	//for backward compatible
	Hash32  = Sum32
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

func Sum32(key []byte) (ret uint32) {
	h := pool32.Get().(*hash32)
	h.Write(key)
	ret = h.Sum32()
	h.Reset()
	pool32.Put(h)
	return
}

func Sum128(key []byte) (ret []byte) {
	h := pool128.Get().(*hash128)
	h.Write(key)
	ret = h.Sum(nil)
	h.Reset()
	pool128.Put(h)
	return
}
