package testutil

import (
	"testing"

	"github.com/interline-io/gotransit"
)

// BenchmarkReader .
func BenchmarkReader(b *testing.B, fe ReaderTester, newReader func() gotransit.Reader) {
	for i := 0; i < b.N; i++ {
		reader := newReader()
		if err := reader.Open(); err != nil {
			b.Error(err)
		}
		fe.Benchmark(b, reader)
		if err := reader.Close(); err != nil {
			b.Error(err)
		}
	}
}

// BenchmarkWriter .
func BenchmarkWriter(b *testing.B, fe ReaderTester, newReader func() gotransit.Reader, newWriter func() gotransit.Writer) {
	for i := 0; i < b.N; i++ {
		// Open Reader
		reader := newReader()
		if err := reader.Open(); err != nil {
			b.Error(err)
		}
		defer reader.Close()
		// Open Writer
		writer := newWriter()
		if err := writer.Open(); err != nil {
			b.Error(err)
		}
		if err := writer.Create(); err != nil {
			b.Error(err)
		}
		// Time only the copy
		b.ResetTimer()
		if err := DirectCopy(reader, writer); err != nil {
			b.Error(err)
		}
		b.StopTimer()
		// Go ahead and run the validations
		r2, err := writer.NewReader()
		if err != nil {
			b.Error(err)
		}
		fe.Benchmark(b, r2)
		if err := writer.Close(); err != nil {
			b.Error(err)
		}
	}
}
