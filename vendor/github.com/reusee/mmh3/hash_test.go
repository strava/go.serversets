package mmh3

import (
	"crypto/rand"
	"fmt"
	"io"
	"testing"
)

func TestHash128(t *testing.T) {
	h := New128()

	cases := map[string]string{
		"":                "00000000000000000000000000000000",
		"hello":           "029bbd41b3a7d8cb191dae486a901e5b",
		"foobar":          "455ac81671aed2bdafd6f8bae055a274",
		"ooooooooooooooo": "a9bd51f7e15176d22148141c49ea8fa5",
		"我能吞下玻璃而不伤身体":     "2ea7aa45a1a1e43d44afaa81c30d1a37",
	}
	for key, hex := range cases {
		h.Write([]byte(key))
		if fmt.Sprintf("%x", h.Sum(nil)) != hex {
			t.Fatal()
		}
		h.Reset()

		for _, c := range key {
			h.Write([]byte(string(c)))
		}
		if fmt.Sprintf("%x", h.Sum(nil)) != hex {
			t.Fatal()
		}
		h.Reset()

		if fmt.Sprintf("%x", Sum128([]byte(key))) != hex {
			t.Fatal()
		}
	}

	// for coverage
	if h.BlockSize() != 16 {
		t.Fatal()
	}
	if h.Size() != 16 {
		t.Fatal()
	}
	h.Sum([]byte{'o'})
}

func bench128(b *testing.B, bytes int) {
	bs := make([]byte, bytes)
	io.ReadFull(rand.Reader, bs)
	b.SetBytes(int64(bytes))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sum128(bs)
	}
}

func BenchmarkHash128_1(b *testing.B)    { bench128(b, 1) }
func BenchmarkHash128_2(b *testing.B)    { bench128(b, 2) }
func BenchmarkHash128_4(b *testing.B)    { bench128(b, 4) }
func BenchmarkHash128_8(b *testing.B)    { bench128(b, 8) }
func BenchmarkHash128_16(b *testing.B)   { bench128(b, 16) }
func BenchmarkHash128_32(b *testing.B)   { bench128(b, 32) }
func BenchmarkHash128_64(b *testing.B)   { bench128(b, 64) }
func BenchmarkHash128_128(b *testing.B)  { bench128(b, 128) }
func BenchmarkHash128_256(b *testing.B)  { bench128(b, 256) }
func BenchmarkHash128_512(b *testing.B)  { bench128(b, 512) }
func BenchmarkHash128_1024(b *testing.B) { bench128(b, 1024) }
func BenchmarkHash128_2048(b *testing.B) { bench128(b, 2048) }
func BenchmarkHash128_4096(b *testing.B) { bench128(b, 4096) }
func BenchmarkHash128_8192(b *testing.B) { bench128(b, 8192) }
