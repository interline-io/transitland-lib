package tl

import "testing"

var result int

func Benchmark_StringToSeconds(b *testing.B) {
	for n := 0; n < b.N; n++ {
		StringToSeconds("12:34:56")
	}
}

func Benchmark_NewWideTime(b *testing.B) {
	for n := 0; n < b.N; n++ {
		NewWideTime("12:34:56")
	}
}
