// Package fixed implements decimal fixed-point numbers.
package fixed

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"unsafe"

	mu "github.com/avdva/numeric/internal/mathutil"
	su "github.com/avdva/numeric/internal/strutil"
)

//go:generate echo $FRAC

const (
	dot   = -8
	scale = 1e8

	bitsInNumber = unsafe.Sizeof(number(0)) * 8

	maxNumber         = 999999999999999999
	minNumber         = -maxNumber
	smallestPosNumber = 1
	smallestNegNumber = -smallestPosNumber
	totalDigits       = 18
)

const (
	Zero             = Fixed(0)
	Max              = Fixed(maxNumber)
	Min              = -Max
	SmallestPositive = Fixed(smallestPosNumber)
	SmallestNegative = Fixed(smallestNegNumber)
)

type number = int64

type Fixed number

func integ(f Fixed) number {
	return number(f) / scale
}

func frac(f Fixed) number {
	return number(f) % scale
}

func fromIntAndFrac(integ int64, frac uint64) Fixed {
	//  9223372036854775807
	// 18446744073709551615
	diff := -dot - mu.DecimalDigits(frac)
	if diff > 0 {
		frac *= uint64(mu.Pow10(diff))
	} else if diff < 0 {
		frac /= uint64(mu.Pow10(-diff))
	}
	result := Fixed(mu.AbsInt64(integ)*scale + int64(frac))
	if integ < 0 {
		result = -result
	}
	return result
}

func FromMantAndExp(mant int64, exp int32) Fixed {
	m, _ := mu.ScaleMant(uint64(mu.AbsInt64(mant)), exp, dot)
	if m > maxNumber {
		m = maxNumber
	}
	if mant < 0 {
		m = -m
	}
	return Fixed(m)
}

func FromString(s string) (Fixed, error) {
	neg, e, digits, err := su.Parse(s)
	if err != nil {
		return Zero, err
	}
	m, e, err := su.FromDigitsAndExp(digits, e, totalDigits)
	if err != nil {
		return Zero, err
	}
	mi := int64(m)
	if neg {
		mi = -mi
	}
	return FromMantAndExp(mi, e), nil
}

func MustFromString(s string) Fixed {
	f, err := FromString(s)
	if err != nil {
		panic(err)
	}
	return f
}

func FromFloat64(f float64) (Fixed, error) {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return Zero, fmt.Errorf("bad float number")
	}
	if f == 0 {
		return Zero, nil
	}
	intMant, e := unit64ToInt64(mu.FloatMantissa(f, 1e-10))
	if math.Signbit(f) {
		intMant = -intMant
	}
	return FromMantAndExp(intMant, e), nil
}

func (f Fixed) Sign() int {
	return mu.Int64Sign(int64(f))
}

func (f Fixed) Abs() Fixed {
	return Fixed(mu.AbsInt64(int64(f)))
}

func (f Fixed) String() string {
	var builder strings.Builder
	su.FormatMantExp(f.Sign(), uint64(f.Abs()), dot, 'f', &builder)
	return builder.String()
}

func (f Fixed) Format(fs fmt.State, c rune) {
	su.FormatMantExp(f.Sign(), uint64(f.Abs()), dot, c, fs)
}

func (f Fixed) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteRune('"')
	if err := su.FormatMantExp(f.Sign(), uint64(f.Abs()), dot, 'e', &b); err != nil {
		return nil, err
	}
	b.WriteRune('"')
	return b.Bytes(), nil
}

func (f *Fixed) UnmarshalJSON(data []byte) error {
	fs, err := FromString(string(data))
	if err == nil {
		*f = fs
	}
	return err
}

func (f Fixed) Cmp(other Fixed) int {
	if f == other {
		return 0
	}
	if f > other {
		return 1
	}
	return -1
}

func (f Fixed) Float64() float64 {
	return float64(f) / scale
}

func (f Fixed) Add(other Fixed) Fixed {
	return Fixed(f + other)
}

func (f Fixed) Sub(other Fixed) Fixed {
	return Fixed(f - other)
}

func (f Fixed) Mul(other Fixed) Fixed {
	a1, a0 := integ(f), frac(f)
	b1, b0 := integ(other), frac(other)
	// TODO(avd) - could produce negative values in the case of overflow
	return Fixed(a1*(b1*scale+b0) + a0*b1 + (a0*b0)/scale)
}

func (f Fixed) Div(other Fixed) Fixed {
	if other == Zero {
		panic("division by zero")
	}
	if f == Zero {
		return Zero
	}
	if q, r, e := mu.QuoRem64(uint64(f.Abs()), dot, uint64(other.Abs()), dot); r == 0 {
		m, e := unit64ToInt64(q, e)
		if !mu.SameSign(int64(f), int64(other)) {
			m = -m
		}
		return FromMantAndExp(m, e)
	}
	res, _ := FromFloat64(float64(f) / float64(other))
	return res
}

func unit64ToInt64(u uint64, e int32) (mant int64, exp int32) {
	if u > math.MaxInt64 {
		u /= 10
		e++
	}
	return int64(u), e
}
