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

func sign(i int64) int {
	if i == 0 {
		return 0
	}
	if i > 0 {
		return 1
	}
	return -1
}
