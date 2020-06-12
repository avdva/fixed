package fixed

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"

	of "github.com/robaho/fixed"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestFromMantAndExp(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		mant int64
		exp  int32
		f    Fixed
	}{
		{0, 0, Zero},
		{1, 0, 1e8},
		{1, -1, 1e7},
		{-1, 0, -1e8},
		{maxNumber, dot, maxNumber},
		{-maxNumber, dot, -maxNumber},
		{1, -8, smallestPosNumber},
		{-1, -8, smallestNegNumber},

		{123456, -3, fromIntAndFrac(123, 456)},
		{-123456, -3, fromIntAndFrac(-123, 456)},
		{math.MaxInt64, dot, Max},
		{-math.MaxInt64, dot, Min},
		{1, -9, Zero},
		{-1, -9, Zero},
		{1, dot, SmallestPositive},
		{-1, dot, SmallestNegative},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(test.f, FromMantAndExp(test.mant, test.exp))
		})
	}
}

func TestFromString(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		s   string
		err string
		f   Fixed
	}{
		{
			"0", "", Zero,
		},
		{
			"-0", "", Zero,
		},
		{
			"-0e0", "", Zero,
		},
		{
			"-123.456", "", fromIntAndFrac(-123, 456),
		},
		{
			"123.456", "", fromIntAndFrac(123, 456),
		},
		{
			"123.456e4", "", fromIntAndFrac(1234560, 0),
		},
		{
			"-123.456e4", "", fromIntAndFrac(-1234560, 0),
		},
		{
			"123.456e-3", "", fromIntAndFrac(0, 1234560),
		},
		{
			"123456789e-9", "", FromMantAndExp(12345678, dot),
		},
		{
			"-123456789e-9", "", FromMantAndExp(-12345678, dot),
		},
		{
			".1234567891011", "", FromMantAndExp(12345678, dot),
		},
		{
			"-.1234567891011", "", FromMantAndExp(-12345678, dot),
		},
		{
			"9999999999999999999999", "", Max,
		},
		{
			"-9999999999999999999999", "", Min,
		},
		{
			"0.00000001", "", SmallestPositive,
		},
		{
			"-0.00000001", "", SmallestNegative,
		},
		{
			"", "empty input", Zero,
		},
		{
			"    -bad", "parsing failed: unexpected symbol 'b' at pos 6", Zero,
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			f, err := FromString(test.s)
			if len(test.err) > 0 {
				a.Panics(func() {
					MustFromString(test.s)
				})
				a.EqualError(err, test.err)
			} else {
				a.Equal(test.f, f)
			}
		})
	}
}

func TestFromFloat64(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		fl  float64
		err string
		f   Fixed
	}{
		{
			0, "", Zero,
		},
		{
			-0, "", Zero,
		},
		{
			-123.456, "", fromIntAndFrac(-123, 456),
		},
		{
			123.456, "", fromIntAndFrac(123, 456),
		},
		{
			9999999999999999999999, "", Max,
		},
		{
			-9999999999999999999999, "", Min,
		},
		{
			0.00000001, "", SmallestPositive,
		},
		{
			maxNumber, "", Max,
		},
		{
			minNumber, "", Min,
		},
		{
			math.NaN(), "bad float number", Zero,
		},
		{
			math.Inf(1), "bad float number", Zero,
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			f, err := FromFloat64(test.fl)
			if len(test.err) > 0 {
				a.EqualError(err, test.err)
			} else {
				a.Equal(test.f, f)
			}
		})
	}
}

func TestSignAbs(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		f    Fixed
		sign int
		abs  Fixed
	}{
		{
			Zero, 0, Zero,
		},
		{
			Max, 1, Max,
		},
		{
			Min, -1, Max,
		},
		{
			SmallestPositive, 1, SmallestPositive,
		},
		{
			SmallestNegative, -1, SmallestPositive,
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			sign, abs := test.f.Sign(), test.f.Abs()
			a.Equal(test.sign, sign)
			a.Equal(test.abs, abs)
			if sign != 0 {
				a.Equal(1, test.abs.Sign())
			}
		})
	}
}

func TestStringConv(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		f          Fixed
		formatF    string
		formatE    string
		formatJSON string
	}{
		{
			Zero, "0", "0", `"0"`,
		},
		{
			fromIntAndFrac(123, 456), "123.456", "123456e-3", `"123456e-3"`,
		},
		{
			fromIntAndFrac(-123, 456), "-123.456", "-123456e-3", `"-123456e-3"`,
		},
		{
			Max, "9999999999.99999999", "999999999999999999e-8", `"999999999999999999e-8"`,
		},
		{
			Min, "-9999999999.99999999", "-999999999999999999e-8", `"-999999999999999999e-8"`,
		},
		{
			SmallestPositive, "0.00000001", "1e-8", `"1e-8"`,
		},
		{
			SmallestNegative, "-0.00000001", "-1e-8", `"-1e-8"`,
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(test.formatF, test.f.String())
			a.Equal(test.formatF, fmt.Sprintf("%f", test.f))
			a.Equal(test.formatE, fmt.Sprintf("%e", test.f))
			if data, err := json.Marshal(test.f); a.NoError(err) {
				a.Equal(test.formatJSON, string(data))
				var f Fixed
				if a.NoError(json.Unmarshal(data, &f)) {
					a.Equal(test.f, f)
				}
			}
		})
	}
}

func TestCmp(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		f1, f2 Fixed
		cmp    int
	}{
		{
			Zero, Zero, 0,
		},
		{
			SmallestPositive, Zero, 1,
		},
		{
			Zero, SmallestNegative, 1,
		},
		{
			Max, Min, 1,
		},
		{
			SmallestPositive, SmallestPositive, 0,
		},
		{
			SmallestNegative, SmallestNegative, 0,
		},
		{
			SmallestPositive, SmallestPositive + SmallestPositive, -1,
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(test.cmp, test.f1.Cmp(test.f2))
			a.Equal(-test.cmp, test.f2.Cmp(test.f1))
		})
	}
}

func TestFloat64(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		f   Fixed
		flt float64
	}{
		{
			Zero, 0,
		},
		{
			fromIntAndFrac(123, 456), 123.456,
		},
		{
			fromIntAndFrac(-123, 456), -123.456,
		},
		{
			SmallestPositive, 0.00000001,
		},
		{
			SmallestNegative, -0.00000001,
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(test.flt, test.f.Float64())
		})
	}
}

func TestAdd(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a, b, result Fixed
	}{
		{
			Zero, Zero, Zero,
		},
		{
			Min, Max, Zero,
		},
		{
			SmallestNegative, SmallestPositive, Zero,
		},
		{
			fromIntAndFrac(123, 456), fromIntAndFrac(123, 456), fromIntAndFrac(246, 912),
		},
		{
			fromIntAndFrac(-123, 456), fromIntAndFrac(-123, 456), fromIntAndFrac(-246, 912),
		},
		{
			fromIntAndFrac(123, 456), fromIntAndFrac(-123, 456), Zero,
		},
		{
			Max, Max, maxNumber + maxNumber,
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(test.result, test.a.Add(test.b))
		})
	}
}

func TestSub(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a, b, result Fixed
	}{
		{
			Zero, Zero, Zero,
		},
		{
			Min, Max, -(maxNumber + maxNumber),
		},
		{
			SmallestNegative, SmallestPositive, 2 * smallestNegNumber,
		},
		{
			fromIntAndFrac(123, 456), fromIntAndFrac(123, 456), Zero,
		},
		{
			fromIntAndFrac(-123, 456), fromIntAndFrac(-123, 456), Zero,
		},
		{
			fromIntAndFrac(123, 456), fromIntAndFrac(-123, 456), fromIntAndFrac(246, 912),
		},
		{
			Max, Max, Zero,
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(test.result, test.a.Sub(test.b))
		})
	}
}

func TestMul(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a, b, result Fixed
	}{
		{
			Zero, Zero, Zero,
		},
		{
			Max, Zero, Zero,
		},
		{
			SmallestNegative, SmallestPositive, Zero,
		},
		{
			fromIntAndFrac(123, 456), fromIntAndFrac(123, 456), fromIntAndFrac(15241, 383936),
		},
		{
			fromIntAndFrac(-123, 456), fromIntAndFrac(123, 456), fromIntAndFrac(-15241, 383936),
		},
		{
			Max, SmallestNegative, fromIntAndFrac(-99, 99999999),
		},
		{
			fromIntAndFrac(-12345, 6789), fromIntAndFrac(-12345, 6789), FromMantAndExp(15241578750190521, -8),
		},
	}
	for i, test := range tests {
		if i != 4 {
			continue
		}
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(test.result, test.a.Mul(test.b))
		})
	}
}

func TestDiv(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a, b, result Fixed
	}{
		{
			Zero, Zero, Zero,
		},
		{
			Max, Zero, Zero,
		},
		{
			SmallestNegative, SmallestPositive, fromIntAndFrac(-1, 0),
		},
		{
			fromIntAndFrac(123, 456), fromIntAndFrac(123, 456), fromIntAndFrac(1, 0),
		},
		{
			fromIntAndFrac(-123, 456), fromIntAndFrac(123, 456), fromIntAndFrac(-1, 0),
		},
		{
			fromIntAndFrac(0, 45), fromIntAndFrac(0, 15), fromIntAndFrac(3, 0),
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if test.b == Zero {
				a.Panics(func() {
					test.a.Div(test.b)
				})
			} else {
				a.Equal(test.result, test.a.Div(test.b))
			}
		})
	}
}

func BenchmarkMulOtherFixed(b *testing.B) {
	f0 := of.NewF(123456789.9)
	f1 := of.NewF(1234.9)

	for i := 0; i < b.N; i++ {
		f0.Mul(f1)
	}
}

func BenchmarkMulFixed(b *testing.B) {
	f0, _ := FromFloat64(123456789.0)
	f1, _ := FromFloat64(1234.0)

	for i := 0; i < b.N; i++ {
		f0.Mul(f1)
	}
}

func BenchmarkMulDecimal(b *testing.B) {
	f0 := decimal.NewFromFloat(123456789.0)
	f1 := decimal.NewFromFloat(1234.0)

	for i := 0; i < b.N; i++ {
		f0.Mul(f1)
	}
}
