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
