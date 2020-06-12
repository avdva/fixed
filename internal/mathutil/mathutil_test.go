package mathutil

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormFloat64(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		f   float64
		res float64
		e   int32
	}{
		{0.012345, 1.2345, 2},
		{12345e50, 1.23455, -54},
		{0, 0, 0},
		{1, 1, 0},
		{10, 10, 0},
		{-5, 0, 0},
		{math.Inf(1), 0, 0},
		{math.Inf(-1), 0, 0},
		{math.NaN(), 0, 0},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			e := normFloat64(test.f)
			f := test.f * math.Pow10(int(e))
			a.Equal(test.e, e)
			if !math.IsInf(test.f, 0) && !math.IsNaN(test.f) {
				a.InDelta(test.res, f, 1e10)
			}
		})
	}
}

func TestMul64(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a, b, res uint64
		exp       int32
	}{
		{math.MaxUint64, 10, math.MaxUint64, -1},
		{math.MaxUint64, 100, math.MaxUint64, -2},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			res, exp := Mul64(test.a, test.b)
			a.Equal(test.res, res)
			a.Equal(test.exp, exp)
		})
	}
}

func TestMulDec(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a, b   uint64
		hi, lo uint64
	}{
		{999999999999999999, 999999999999999999, 999999999999999998, 000000000000000001},
		{123456789101214, 98765432100, 12193263, 121259971344569400},
		{123456789, 987654321, 0, 123456789 * 987654321},
		{12345600000, 12345600000, 152, 413839360000000000},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			hi, lo := MulDec(test.a, test.b)
			a.Equal(test.hi, hi, "%d", hi)
			a.Equal(test.lo, lo, "%d", lo)
		})
	}
}

func TestShrDec(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a, b     uint64
		decimals int
		hi, lo   uint64
		panics   bool
	}{
		{math.MaxUint64, 0, 0, 0, 0, true},
		{0, math.MaxUint64, 0, 0, 0, true},
		{1234, 5678, -1, 0, 0, true},
		{999999999999999999, 9999999999999999999, 3, 999999999999999, 9999999999999999999, false},
		{999999999999999999, 9999999999999999999, 18, 0, 9999999999999999999, false},
		{999999999999999999, 999999999999999999, 19, 0, 999999999999999999, false},

		{maxDecUint64, maxDecUint64, 1, maxDecUint64 / 10, (maxDecUint64%10)*1e18 + maxDecUint64/10, false},
		{0, 123456789, 5, 0, 1234, false},
		{maxDecUint64, 0, 6, 9999999999999, 9999990000000000000, false},
		{maxDecUint64, 0, 19, 0, maxDecUint64, false},
		{maxDecUint64, 0, 20, 0, maxDecUint64 / 10, false},
		{152, 413839360000000000, 8, 0, 0, false},
	}
	for i, test := range tests {
		if i != len(tests)-1 {
			continue
		}
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if test.panics {
				a.Panics(func() {
					SrhDec(test.a, test.b, test.decimals)
				})
				return
			}
			hi, lo := SrhDec(test.a, test.b, test.decimals)
			a.Equal(test.hi, hi, "%d", hi)
			a.Equal(test.lo, lo, "%d", lo)
		})
	}
}

func BenchmarkInt64Sign(b *testing.B) {
	var dummy int
	for i := 0; i < b.N; i++ {
		dummy += Int64Sign(int64(i)) + Int64Sign(int64(-i)) + Int64Sign(int64(i-i))
	}
	// this metric is just to prevent unwanted optimisations in calculations of `dummy.`
	b.ReportMetric(float64(dummy), "dummy_metric")
}

func BenchmarkIfSign(b *testing.B) {
	var dummy int
	for i := 0; i < b.N; i++ {
		dummy += sign(int64(i)) + sign(int64(-i)) + sign(int64(i-i))
	}
	// this metric is just to prevent unwanted optimisations in calculations of `dummy.`
	b.ReportMetric(float64(dummy), "dummy_metric")
}

func BenchmarkMul64(b *testing.B) {
	var dummy float64
	for i := 0; i < b.N; i++ {
		m, e := Mul64(uint64(i*1e10), uint64(i*1e10))
		dummy += (float64(m) + float64(e))
	}
	// this metric is just to prevent unwanted optimisations in calculations of `dummy.`
	b.ReportMetric(dummy, "dummy_metric")
}

func BenchmarkMulDec64(b *testing.B) {
	var dummy float64
	for i := 0; i < b.N; i++ {
		m, e := MulDec64(uint64(i*1e10), uint64(i*1e10))
		dummy += (float64(m) + float64(e))
	}
	// this metric is just to prevent unwanted optimisations in calculations of `dummy.`
	b.ReportMetric(dummy, "dummy_metric")
}

func sign(i int64) int {
	if i == 0 {
		return 0
	}
	if i > 0 {
		return 1
	}
	return -1
}
