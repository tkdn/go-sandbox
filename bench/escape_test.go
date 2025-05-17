package escape_test

import "testing"

// このベンチマークは Go 100Tips における
// No.95 Stack, Heap の違いを理解していないのサンプル

var globalValue int

func sumPtr(x, y int) *int {
	z := x + y
	return &z
}

func sumVal(x, y int) int {
	z := x + y
	return z
}

func BenchmarkSumPtr(b *testing.B) {
	b.ReportAllocs()
	var local *int
	var i int
	for b.Loop() {
		i++
		local = sumPtr(i, i)
	}
	globalValue = *local
}

func BenchmarkSumVal(b *testing.B) {
	b.ReportAllocs()
	var local int
	var i int
	for b.Loop() {
		i++
		local = sumVal(i, i)
	}
	globalValue = local
}
