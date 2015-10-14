package mmh3

import (
	"crypto/rand"
	"fmt"
	"io"
	"testing"
)

func TestHash32(t *testing.T) {
	h := New32()

	cases := map[string]string{
		"":            "00000000",
		"hello":       "47fa8b24",
		"foobar":      "bdd4c4a4",
		"ooooooo":     "cc77ff34",
		"我能吞下玻璃而不伤身体": "841a69c4",
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
	}

	cases2 := map[string]uint32{
		"":            0,
		"hello":       613153351,
		"foobar":      2764362941,
		"ooooooo":     889157580,
		"我能吞下玻璃而不伤身体": 3295222404,
	}
	for key, hash := range cases2 {
		h.Write([]byte(key))
		if h.Sum32() != hash {
			t.Fatal()
		}
		h.Reset()

		if Sum32([]byte(key)) != hash {
			t.Fatal()
		}
	}

	// for coverage
	if h.BlockSize() != 4 {
		t.Fatal()
	}
	if h.Size() != 4 {
		t.Fatal()
	}
	h.Sum([]byte{'o'})
}

func bench32(b *testing.B, bytes int) {
	bs := make([]byte, bytes)
	io.ReadFull(rand.Reader, bs)
	b.SetBytes(int64(bytes))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sum32(bs)
	}
}

func BenchmarkHash32_1(b *testing.B)    { bench32(b, 1) }
func BenchmarkHash32_2(b *testing.B)    { bench32(b, 2) }
func BenchmarkHash32_4(b *testing.B)    { bench32(b, 4) }
func BenchmarkHash32_8(b *testing.B)    { bench32(b, 8) }
func BenchmarkHash32_16(b *testing.B)   { bench32(b, 16) }
func BenchmarkHash32_32(b *testing.B)   { bench32(b, 32) }
func BenchmarkHash32_64(b *testing.B)   { bench32(b, 64) }
func BenchmarkHash32_128(b *testing.B)  { bench32(b, 128) }
func BenchmarkHash32_256(b *testing.B)  { bench32(b, 256) }
func BenchmarkHash32_512(b *testing.B)  { bench32(b, 512) }
func BenchmarkHash32_1024(b *testing.B) { bench32(b, 1024) }
func BenchmarkHash32_2048(b *testing.B) { bench32(b, 2048) }
func BenchmarkHash32_4096(b *testing.B) { bench32(b, 4096) }
func BenchmarkHash32_8192(b *testing.B) { bench32(b, 8192) }
