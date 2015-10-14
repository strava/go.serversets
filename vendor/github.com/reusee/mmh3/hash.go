package mmh3

import (
	"bytes"
	"encoding/binary"
	"hash"
	"reflect"
	"unsafe"
)

const (
	c1_64 uint64 = 0x87c37b91114253d5
	c2_64 uint64 = 0x4cf5ad432745937f
)

type hash128 struct {
	h1   uint64
	h2   uint64
	tail []byte
	size uint64
}

func New128() hash.Hash {
	return new(hash128)
}

func (h *hash128) BlockSize() int {
	return 16
}

func (h *hash128) Reset() {
	h.h1 = 0
	h.h2 = 0
	h.tail = nil
	h.size = 0
}

func (h *hash128) Size() int {
	return 16
}

func (h *hash128) Sum(in []byte) []byte {
	var k1, k2 uint64
	h1 := h.h1
	h2 := h.h2
	tail := h.tail

	if tail != nil {
		switch len(h.tail) {
		case 15:
			k2 ^= uint64(tail[14]) << 48
			fallthrough
		case 14:
			k2 ^= uint64(tail[13]) << 40
			fallthrough
		case 13:
			k2 ^= uint64(tail[12]) << 32
			fallthrough
		case 12:
			k2 ^= uint64(tail[11]) << 24
			fallthrough
		case 11:
			k2 ^= uint64(tail[10]) << 16
			fallthrough
		case 10:
			k2 ^= uint64(tail[9]) << 8
			fallthrough
		case 9:
			k2 ^= uint64(tail[8])
			k2 *= c2_64
			k2 = (k2 << 33) | (k2 >> (64 - 33))
			k2 *= c1_64
			h2 ^= k2
			fallthrough
		case 8:
			k1 ^= uint64(tail[7]) << 56
			fallthrough
		case 7:
			k1 ^= uint64(tail[6]) << 48
			fallthrough
		case 6:
			k1 ^= uint64(tail[5]) << 40
			fallthrough
		case 5:
			k1 ^= uint64(tail[4]) << 32
			fallthrough
		case 4:
			k1 ^= uint64(tail[3]) << 24
			fallthrough
		case 3:
			k1 ^= uint64(tail[2]) << 16
			fallthrough
		case 2:
			k1 ^= uint64(tail[1]) << 8
			fallthrough
		case 1:
			k1 ^= uint64(tail[0])
			k1 *= c1_64
			k1 = (k1 << 31) | (k1 >> (64 - 31))
			k1 *= c2_64
			h1 ^= k1
		}
	}

	h1 ^= uint64(h.size)
	h2 ^= uint64(h.size)
	h1 += h2
	h2 += h1
	h1 ^= h1 >> 33
	h1 *= 0xff51afd7ed558ccd
	h1 ^= h1 >> 33
	h1 *= 0xc4ceb9fe1a85ec53
	h1 ^= h1 >> 33
	h2 ^= h2 >> 33
	h2 *= 0xff51afd7ed558ccd
	h2 ^= h2 >> 33
	h2 *= 0xc4ceb9fe1a85ec53
	h2 ^= h2 >> 33
	h1 += h2
	h2 += h1

	h.h1 = h1
	h.h2 = h2

	ret := make([]byte, 16)
	retHeader := (*reflect.SliceHeader)(unsafe.Pointer(&ret))
	var tuple []uint64
	tupleHeader := (*reflect.SliceHeader)(unsafe.Pointer(&tuple))
	tupleHeader.Data = retHeader.Data
	tupleHeader.Len = 2
	tupleHeader.Cap = 2
	tuple[0] = h1
	tuple[1] = h2

	if in == nil {
		return ret
	}
	return append(in, ret...)

}

func (h *hash128) Write(key []byte) (n int, err error) {
	n = len(key)
	h.size += uint64(n)
	h1 := h.h1
	h2 := h.h2

	if h.tail != nil {
		n := 16 - len(h.tail)
		if n > len(key) {
			n = len(key)
		}
		h.tail = append(h.tail, key[:n]...)
		key = key[n:]
		if len(h.tail) == 16 { // a full block
			var k1, k2 uint64
			r := bytes.NewReader(h.tail)
			binary.Read(r, binary.LittleEndian, &k1)
			binary.Read(r, binary.LittleEndian, &k2)
			k1 *= c1_64
			k1 = (k1 << 31) | (k1 >> (64 - 31))
			k1 *= c2_64
			h1 ^= k1
			h1 = (h1 << 27) | (h1 >> (64 - 27))
			h1 += h2
			h1 = h1*5 + 0x52dce729
			k2 *= c2_64
			k2 = (k2 << 33) | (k2 >> (64 - 33))
			k2 *= c1_64
			h2 ^= k2
			h2 = (h2 << 31) | (h2 >> (64 - 31))
			h2 += h1
			h2 = h2*5 + 0x38495ab5
			h.tail = nil
		}
	}

	length := len(key)
	nblocks := length / 16
	if nblocks > 0 {
		var k1, k2 uint64
		var blocks [][2]uint64
		keyHeader := (*reflect.SliceHeader)(unsafe.Pointer(&key))
		blocksHeader := (*reflect.SliceHeader)(unsafe.Pointer(&blocks))
		blocksHeader.Data = keyHeader.Data
		blocksHeader.Len = nblocks
		blocksHeader.Cap = nblocks
		for _, b := range blocks {
			k1, k2 = b[0], b[1]
			k1 *= c1_64
			k1 = (k1 << 31) | (k1 >> (64 - 31))
			k1 *= c2_64
			h1 ^= k1
			h1 = (h1 << 27) | (h1 >> (64 - 27))
			h1 += h2
			h1 = h1*5 + 0x52dce729
			k2 *= c2_64
			k2 = (k2 << 33) | (k2 >> (64 - 33))
			k2 *= c1_64
			h2 ^= k2
			h2 = (h2 << 31) | (h2 >> (64 - 31))
			h2 += h1
			h2 = h2*5 + 0x38495ab5
		}
	}

	if length%16 != 0 {
		h.tail = key[nblocks*16 : length]
	}

	h.h1 = h1
	h.h2 = h2
	return
}
