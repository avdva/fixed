package mathutil

import (
	"math"
	"math/bits"
	"unsafe"
)

var (
	decimalFactorTable = [...]uint64{ // up to 1e19
		1, 10, 100, 1000, 10000,
		100000, 1000000, 10000000, 100000000, 1000000000, 10000000000,
		100000000000, 1000000000000, 10000000000000, 100000000000000,
		1000000000000000, 10000000000000000, 100000000000000000,
		1000000000000000000, 10000000000000000000,
	}

	digitsHelper = [...]int{
		0, 0, 0, 0, 1, 1, 1, 2, 2, 2,
		3, 3, 3, 3, 4, 4, 4, 5, 5, 5,
		6, 6, 6, 6, 7, 7, 7, 8, 8, 8,
		9, 9, 9, 9, 10, 10, 10, 11, 11, 11,
		12, 12, 12, 12, 13, 13, 13, 14, 14, 14,
		15, 15, 15, 15, 16, 16, 16, 17, 17, 17,
		18, 18, 18, 18, 19,
	}
)

// Pow10 returns 10^pow.
func Pow10(pow int) uint64 {
	if pow < 0 || pow >= len(decimalFactorTable) {
		return 0
	}
	return decimalFactorTable[pow]
}

func BinaryDigits(value uint64) int {
	return int(8*unsafe.Sizeof(uint64(0))) - bits.LeadingZeros64(value)
}

// DecimalDigits returns the number of decimal digits in 'value'.
// see https://stackoverflow.com/a/25934909
func DecimalDigits(value uint64) int {
	if value == 0 {
		return 1
	}

	digits := digitsHelper[BinaryDigits(value)]
	if value >= decimalFactorTable[digits] {
		digits++
	}
	return digits
}

func DecimalLenInt64(value int64) int {
	result := 0
	if value < 0 {
		result++
		value = -value
	}
	return result + DecimalDigits(uint64(value))
}

// Mul64 performs a multiplication of two 64-bit values.
// If the result exceeds 64 bits, the function returns (a*b)/10^(-exp),
//	where exp is the minumum possible value, so that the result fits 64 bits.
func Mul64(a, b uint64) (mant uint64, exp int32) {
	hi, lo := bits.Mul64(a, b)
	if hi > 0 {
		// the result overflows uint64, so we'll divide it by a factor of 10,
		// so that it fits a uint64 value again, and add that factor to the resulting exponent.
		dd := DecimalDigits(hi)
		lo, _ = bits.Div64(hi, lo, Pow10(dd))
		exp = int32(-dd)
	}
	return lo, exp
}

func QuoRem64(m1 uint64, e1 int32, m2 uint64, e2 int32) (quo, rem uint64, e int32) {

	// a*10^e1 / b*10^e2 = (a/b) * 10^(e1-e2)
	e = e1 - e2

	if m1%m2 == 0 {
		return m1 / m2, 0, e
	}

	// give it best chances to division.
	// shift m1 close to the maximum possible number.
	expInc := Log10(math.MaxUint64 / m1)
	e -= int32(expInc)
	m1 *= Pow10(expInc)

	return m1 / m2, m1 % m2, e
}

func Log10(a uint64) int {
	return DecimalDigits(a) - 1
}

func ScaleMant(mant uint64, exp, targetExp int32) (m uint64, exact bool) {
	if mant == 0 {
		return 0, true
	}
	diff := int(exp - targetExp)
	if diff == 0 {
		return mant, true
	}
	p := Pow10(AbsInt(diff))
	if p == 0 {
		if diff > 0 {
			return math.MaxUint64, false
		}
		return 0, false
	}
	if diff > 0 {
		if math.MaxUint64/mant < p {
			return math.MaxUint64, false
		}
		mant, exact = mant*p, true
	} else {
		mant, exact = mant/p, mant%p == 0
	}
	return mant, exact
}

func AbsInt(val int) int {
	mask := val >> (unsafe.Sizeof(int(0))*8 - 1)
	return (val + mask) ^ mask
}

func AbsInt64(val int64) int64 {
	mask := val >> (unsafe.Sizeof(int64(0))*8 - 1)
	return (val + mask) ^ mask
}

func SameSign(a, b int64) bool {
	return (a>>63 ^ b>>63) == 0
}

// normFloat64 calculates such e, that 1 <= abs(f)*(10**e) <= 10
func normFloat64(f float64) (exp int32) {
	if f == 0 || math.IsInf(f, 0) || math.IsNaN(f) {
		return 0
	}
	f = math.Abs(f)
	switch {
	case f < 1:
		exp = int32(math.Log10(1/f)) + 1
	case f > 10:
		exp = -(int32(math.Log10(f/10)) + 1)
	default:
		return 0
	}
	return exp
}

// FloatMantissa returns such (mant, e) that abs(mant*(10^-e) - f) < epsilon.
func FloatMantissa(f float64, epsilon float64) (mant uint64, exp int32) {
	const maxPrec = 19
	var result uint64
	f = math.Abs(f)
	i, exp := int32(0), normFloat64(f)
	for ; ; i++ {
		integ, frac := math.Modf(f * math.Pow10(int(exp+i)))
		result = uint64(integ)
		if frac < epsilon || i >= maxPrec {
			break
		}
	}
	return result, -(exp + i)
}

func TrimMantExp(m uint64, e, eMax int32) (uint64, int32) {
	for e < eMax && m > 9 && m%10 == 0 {
		m /= 10
		e++
	}
	return m, e
}

func Int64Sign(v int64) int {
	if v == 0 {
		return 0
	}
	return [...]int{1, -1}[uint64(v)>>63]
}

func SrhDec(hi, lo uint64, decimals int) (uint64, uint64) {
	if decimals >= 38 {
		return 0, 0
	}
	exp := Pow10(decimals)
	lo /= exp
	if hi > 0 {
		t := hi % exp
		hi /= exp
		if decimals <= 19 {
			lo += t * Pow10(19-decimals)
		} else {
			lo += t / Pow10(decimals-19)
		}
	}
	return hi, lo
}

func MulDec(x, y int64) (hi, lo int64) {
	a, b := x/1e9, x%1e9
	c, d := y/1e9, y%1e9
	lo = b * d
	if a|c == 0 {
		return
	}
	hi = a * c
	t := a*d + c*b
	t1, t2 := t/1e9, t%1e9
	lo += t2 * 1e9
	if carry := lo / 1e18; carry > 0 {
		hi += t1 + carry
		lo = lo % 1e18
	}
	return
}
