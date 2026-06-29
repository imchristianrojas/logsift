package main

import (
	"runtime"
	"testing"

	"github.com/imchristianrojas/logsift/internal/pool"
)

func BenchmarkSequential(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := runSequential("testdata/big.log")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcurrent(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := pool.Run("testdata/big.log", runtime.NumCPU())
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkChunked(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := pool.RunChunked("testdata/big.log", runtime.NumCPU())
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkChunkedSizes sweeps the block size to show the overhead-vs-balance
// curve: tiny blocks pay per-send overhead, huge blocks starve the workers.
func BenchmarkChunkedSizes(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"64KiB", 64 << 10},
		{"256KiB", 256 << 10},
		{"1MiB", 1 << 20},
		{"4MiB", 4 << 20},
		{"16MiB", 16 << 20},
	}
	for _, s := range sizes {
		b.Run(s.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := pool.RunChunkedSize("testdata/big.log", runtime.NumCPU(), s.size)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
