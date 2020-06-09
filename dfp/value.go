// Copyright 2020 Aleksandr Demakin. All rights reserved.

// Package dfp implements a decimal floating-point number, where both mantissa
// and exponent are stored in a single number.
// Can be used to represent currency rates with up to 16 digits of precision.
package dfp

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unsafe"

	mu "github.com/avdva/numeric/internal/mathutil"
	su "github.com/avdva/numeric/internal/strutil"
)

var (
	// JSONMode defines the way all values are marshaled into json.
	// Either Format* constants or JSONModeCompact can be used.
	// This variable is not thread-safe, so this should not be changed concurently.
	JSONMode = JSONModeCompact
)

const (
	// FormatString marshals values as strings, like `"1234.5678"`
	FormatString = iota
	// FormatFloat marshals values as floats, like `1234.5678`.
	FormatFloat
	// FormatJSONObject marshals values with mantissa and exponent, like `{"m":123,"e":-5}`.
	FormatJSONObject

	// JSONModeCompact will choose the shortest form between FormatString and FormatJSONObject.
	JSONModeCompact = -1
)

var (
	jsonParts = []string{`{"m":`, `,"e":`, `}`}
	jsonLen   = len(jsonParts[0]) + len(jsonParts[1]) + len(jsonParts[2])

	digitsInMaxMantissa = mu.DecimalDigits(maxMantissa)
)

const (
	bitsInNumber = unsafe.Sizeof(number(0)) * 8
	mantBits     = bitsInNumber - expBits - signBit

	// maxMantissa is 72057594037927935 for a (8,56) number
	maxMantissa = mantMask
	minMantissa = 1
	maxNumber   = 1<<bitsInNumber - 1
)

const (
	modeFloor = iota
	modeRound
	modeCeil
)

var (
	// Max is the maximum possible fixed-point value.
	Max = combine(maxMantissa, maxExponent)
	// Min is the minimum possible fixed-point value.
	Min = combine(minMantissa, minExponent)
)

type (
	number  = uint64
	expType = int32
)

// Value is a positive decimal floating-point number.
// It currently uses a uint64 value as a data type, where
// 8 bits are used for exponent and 56 for mantissa.
//   63      55                                                     0
//   ________|_______________________________________________________
//   eeeeeeeemmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmm
//
// Value can be useful for representing numbers like prices in financial services.
type Value number

// FromUint64 returns a value for given uint64 number.
// If the number cannot be precisely represented, the least significant digits will be truncated.
func FromUint64(v uint64) Value {
	return adjustMantExp(number(v), 0)
}

// FromMantAndExp returns a value for given mantissa and exponent.
// If the number cannot be precisely represented, the least significant digits will be truncated.
func FromMantAndExp(mant uint64, exp int32) Value {
	return adjustMantExp(number(mant), exp).Normalized()
}

// FromFloat64 returns a value for given float64 value.
// Returns an error for nagative values, infinities, and not-a-numbers.
func FromFloat64(v float64) (Value, error) {
	if v < 0 || math.IsInf(v, 0) || math.IsNaN(v) {
		return zero, fmt.Errorf("bad float number")
	}
	if v == 0 {
		return zero, nil
	}
	mant, e := mu.FloatMantissa(v, 1e-10)
	return adjustMantExp(number(mant), expType(e)).Normalized(), nil
}

// MustFromFloat64 returns a value for given float64 value. It panics on an error.
func MustFromFloat64(f float64) Value {
	v, err := FromFloat64(f)
	if err != nil {
		panic(err)
	}
	return v
}

// FromString parses a string into a value.
func FromString(s string) (Value, error) {
	neg, e, digits, err := su.Parse(s)
	if neg {
		return zero, fmt.Errorf("negative value")
	}
	if err != nil { // could still be a float
		if f, fltErr := strconv.ParseFloat(s, 64); fltErr == nil {
			return FromFloat64(f)
		}
		return zero, err
	}
	m, e, err := su.FromDigitsAndExp(digits, e, digitsInMaxMantissa)
	if err != nil {
		panic(err) // should not normally happen
	}
	return adjustMantExp(number(m), expType(e)), nil
}

// MustFromString parses a string into a value. It panics on an error.
func MustFromString(s string) Value {
	v, err := FromString(s)
	if err != nil {
		panic(err)
	}
	return v
}

// MarshalJSON marshals value according to current JSONMode.
// See JSONMode and JSONMode* constants.
func (v Value) MarshalJSON() ([]byte, error) {
	return v.toJSON(JSONMode), nil
}

func (v Value) toJSON(mode int) []byte {
	switch mode {
	case FormatFloat:
		return []byte(strconv.FormatFloat(v.Float64(), 'f', -1, 64))
	case FormatJSONObject:
		return []byte(toMEJSON(v))
	case JSONModeCompact:
		if decimalFormatLen(v)+2 <= jsonMEFormatLen(v) { // +2 for a pair of quotes
			return v.toJSON(FormatString)
		}
		return v.toJSON(FormatJSONObject)
	default: // marshal as a string
		var builder strings.Builder
		builder.WriteRune('"')
		m, e := split(v)
		su.FormatMantExp(1, uint64(m), int32(e), 'f', &builder)
		builder.WriteRune('"')
		return []byte(builder.String())
	}
}

func toMEJSON(v Value) string {
	var builder strings.Builder
	m, e := split(v)
	builder.WriteString(jsonParts[0])
	builder.WriteString(strconv.FormatUint(m, 10))
	builder.WriteString(jsonParts[1])
	builder.WriteString(strconv.FormatInt(int64(e), 10))
	builder.WriteString(jsonParts[2])
	return builder.String()
}

// UnmarshalJSON unmarshals a string, float, or an object into a value.
func (v *Value) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty json")
	}
	switch data[0] {
	case '{':
		d := struct {
			M number
			E expType
		}{}
		if err := json.Unmarshal(data, &d); err != nil {
			return err
		}
		*v = FromMantAndExp(d.M, d.E)
	default:
		value, err := FromString(string(data))
		if err != nil {
			return err
		}
		*v = value
	}
	return nil
}

// GoString returns debug string representation.
func (v Value) GoString() string {
	m, e := split(v)
	return fmt.Sprintf("{%v, %v}", m, e)
}

// String returns a string representation of the value.
func (v Value) String() string {
	var builder strings.Builder
	m, e := split(v.Normalized())
	su.FormatMantExp(1, uint64(m), int32(e), 'f', &builder)
	return builder.String()
}

// Format implements fmt.Formatter and allows to format values as a string.
//	'f', 's' will produce a decimal string, e.g. 123.456
//	'e', 'v' will produce scientific notation, e.g. 123456e7
func (v Value) Format(f fmt.State, c rune) {
	m, e := split(v.Normalized())
	su.FormatMantExp(1, uint64(m), int32(e), c, f)
}

// MantUint64 returns v's mantissa as is.
func (v Value) MantUint64() uint64 {
	return uint64(mant(v))
}

// Exp returns v's exonent as is.
func (v Value) Exp() int32 {
	return int32(exp(v))
}

// Eq returns true if both values represent the same number.
func (v Value) Eq(other Value) bool {
	if v == other {
		return true
	}
	return v.Normalized() == other.Normalized()
}

// IsZero returns true if the value has zero mantissa.
func (v Value) IsZero() bool {
	return mant(v) == 0
}

// ScaleMant changes the mantissa of v so, that v = m * 10e'exp'.
// As a result, mantissa can lose some digits in precision, become zero, or Max.
// Note, that is some cases the result may be unexpected:
// Consider scaling (m = 1, e = 20) --> e = 0:
//	in this case  mantissa overflows the maximum possible value, and the result will be
//	max_possible_mantissa, 0 which is not equal to the initial value.
func (v Value) ScaleMant(exp int32) (value Value, exact bool) {
	exact = true
	switch {
	case exp < minExponent:
		exp = minExponent
		exact = false
	case exp > maxExponent:
		exp = maxExponent
		exact = false
	case v.IsZero():
		return combine(0, exp), true
	}
	m, e := split(v)
	m, exactScale := mu.ScaleMant(uint64(m), int32(e), exp)
	if m > maxMantissa {
		m, exactScale = maxMantissa, false
	}
	return combine(m, exp), exact && exactScale
}

// Uint64 returns the value as a uint64 number.
func (v Value) Uint64() (mant uint64, exact bool) {
	m, e := split(v.Normalized())
	m, exact = mu.ScaleMant(m, e, 0)
	if m > maxMantissa {
		m, exact = maxMantissa, false
	}
	return m, exact
}

// Float64 returns a float64 value.
func (v Value) Float64() float64 {
	m, e := split(v)
	return float64(m) * math.Pow10(int(e))
}

// Normalized eliminates trailing zeros in the mantissa.
// The process increases the exponent, and stops if it exceeds the maximum possible exponent,
// so that it is possible, that the mantissa will still have trailing zeros.
func (v Value) Normalized() Value {
	m, e := split(v)
	if m == 0 {
		return zero
	}
	return combine(mu.TrimMantExp(m, e, maxExponent))
}

// Cmp compares two values.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func (v Value) Cmp(other Value) int {
	m1, e1 := split(v)
	m2, e2 := split(other)
	ediff := int(e1 - e2)

	if ediff == 0 || m1 == 0 || m2 == 0 {
		return uint64Cmp(m1, m2)
	}

	// compare highest decimal digit position
	maxDigit1 := int(e1) + mu.DecimalDigits(m1)
	maxDigit2 := int(e2) + mu.DecimalDigits(m2)
	if maxDigit1 > maxDigit2 {
		return 1
	} else if maxDigit1 < maxDigit2 {
		return -1
	}

	if ediff > 0 {
		m1 *= mu.Pow10(ediff)
	} else {
		m2 *= mu.Pow10(-ediff)
	}
	return uint64Cmp(m1, m2)
}

// Floor returns the nearest value less than or equal to v that has prec decimal places.
// Note that prec can be negative.
func (v Value) Floor(prec int) Value {
	m, e := split(v)
	return adjustMantExp(round(m, e, prec, modeFloor))
}

// Round rounds the value to prec decimal places.
// Note that prec can be negative.
func (v Value) Round(prec int) Value {
	m, e := split(v)
	return adjustMantExp(round(m, e, prec, modeRound))
}

// Ceil returns the nearest value greater than or equal to v that has prec decimal places.
// Note that prec can be negative.
func (v Value) Ceil(prec int) Value {
	m, e := split(v)
	return adjustMantExp(round(m, e, prec, modeCeil))
}

func round(n number, e expType, prec, mode int) (number, expType) {
	toCut, ei := 0, int(e)
	dd := mu.DecimalDigits(n)
	if prec >= 0 {
		toCut = -ei - prec
	} else {
		if e < 0 {
			dd += ei
			if dd <= 0 { // no digits before the decimal point
				if n > 0 && mode == modeCeil && prec >= 0 {
					return 1, expType(-prec)
				}
				return 0, 0
			}
			n /= mu.Pow10(-ei)
			ei = 0
		}
		toCut = -prec
	}
	if toCut <= 0 {
		return n, e
	}
	if toCut > dd {
		toCut = dd
	}
	p := mu.Pow10(toCut)
	r := n % p
	n /= p
	ei += toCut
	switch mode {
	case modeRound:
		if r > p/2 {
			n++
		}
	case modeCeil:
		if r != 0 {
			if n > 0 {
				n++
			} else if prec >= 0 {
				n = 1
				ei = -prec
			}
		}
	}
	return n, expType(ei)
}

// Add returns the sum of two values.
// If the resulting mantissa overflows max mantissa, the least significant digits will be truncated.
// If the result overflows Max, Max is returned.
func (v Value) Add(other Value) Value {
	m1, e1 := split(v)
	m2, e2 := split(other)
	// first, check for obvious cases, when one of the arguments is zero
	if m1 == 0 {
		if m2 == 0 {
			return zero
		}
		return other
	}
	if m2 == 0 {
		return v
	}
	return addWithExp(toEqualExp(m1, e1, m2, e2))
}

// Sub returns |a-b| and a boolean flag indicating that the result is negative.
func (v Value) Sub(other Value) (Value, bool) {
	m1, e1 := split(v)
	m2, e2 := split(other)
	// first, check for obvious cases, when one of the arguments is zero
	if m2 == 0 {
		if m1 == 0 {
			return zero, false
		}
		return v, false
	}
	if m1 == 0 {
		return other, true
	}
	return subWithExp(toEqualExp(m1, e1, m2, e2))
}

// Mul returns v * other.
// If the result underflows Min, zero is returned.
// If the rsult overflows Max, Max is returned.
// If the resulting mantissa overflows max mantissa, the least significant digits will be truncated.
func (v Value) Mul(other Value) Value {

	v, other = v.Normalized(), other.Normalized()

	m1, e1 := split(v)
	m2, e2 := split(other)

	// first, check for obvious cases, when one of the arguments is zero
	if m1 == 0 || m2 == 0 {
		return zero
	}

	// a*10^e1 * b*10^e2 = a * b * 10^(e1+e2)
	e := int32(e1) + int32(e2)
	// perform a 128-bit multiplication
	res, eShift := mu.Mul64(uint64(m1), uint64(m2))
	e -= eShift

	return adjustMantExp(res, expType(e))
}

// DivMod calculates such quo and rem, that a = b * quo + rem. If b == 0, Div panics.
// Quo will be rounded to prec digits.
// Notice that prec can be negative.
func (v Value) DivMod(other Value, prec int) (quo, rem Value) {

	v, other = v.Normalized(), other.Normalized()
	q, r, e := quoRem(v, other)
	if r == 0 {
		return adjustMantExp(q, e).Normalized(), zero
	}

	if e < maxExponent {
		q, e = mu.TrimMantExp(q, expType(e), maxExponent)
	}

	q, e = round(q, e, prec, modeFloor)
	quo = adjustMantExp(q, expType(e))
	rem, _ = v.Sub(other.Mul(quo))
	return quo, rem
}

// Div calculates a/b. If b == 0, Div panics.
// First, it tries to perform integer division, and if the remainder is zero, returns the result.
// Otherwise, it returns the result of a float64 division.
func (v Value) Div(other Value) Value {

	v, other = v.Normalized(), other.Normalized()

	if quo, rem, e := quoRem(v, other); rem == 0 {
		return adjustMantExp(quo, expType(e)).Normalized()
	}

	return float64Div(v, other)
}

func float64Div(a, b Value) Value {
	m1, e1 := split(a)
	m2, e2 := split(b)

	if m2 == 0 {
		panic("division by zero")
	}

	flt := float64(m1) / float64(m2) * math.Pow10(int(e1)-int(e2))
	result, _ := FromFloat64(flt)
	return result
}

func quoRem(v1, v2 Value) (quo, rem number, e expType) {

	m1, e1 := split(v1)
	m2, e2 := split(v2)

	if m2 == 0 {
		panic("division by zero")
	}
	if m1 == 0 {
		return 0, 0, 0
	}

	q, r, ei := mu.QuoRem64(uint64(m1), int32(e1), uint64(m2), int32(e2))

	return number(q), number(r), expType(ei)
}

// toEqualExp changes m1 and m2 in such a way, that e1 == e2.
// the result can be used to calculate m1+m2, m1-m2.
// if the difference between the exponents is too big, m2 can lose some (or all) digits.
func toEqualExp(m1 number, e1 expType, m2 number, e2 expType) (r1, r2 number, re expType) {

	if e1 >= e2 {
		return doToEqualExp(m1, e1, m2, e2)
	}

	r1, r2, re = doToEqualExp(m2, e2, m1, e1)
	return r2, r1, re
}

// doToEqualExp is a helper for toEqualExp. it assumes that e1 >= e2.
func doToEqualExp(m1 number, e1 expType, m2 number, e2 expType) (r1, r2 number, re expType) {
	ediff := e1 - e2
	if ediff == 0 {
		return m1, m2, e1
	}

	// try to trim trailing zeros for m2.
	// if we have enough to increase e2 so that it equals e1, return the sum.
	m2, e2 = mu.TrimMantExp(m2, e2, e1)
	if ediff = e1 - e2; ediff == 0 {
		return m1, m2, e1
	}

	// next, try to increase m1 and decrease e1 so, that e1 == e2.
	toMult := expType(mu.Log10(maxMantissa / m1))
	if toMult > ediff {
		toMult = ediff
	}
	m1 *= mu.Pow10(int(toMult))
	e1 -= toMult

	if ediff = e1 - e2; ediff == 0 {
		return m1, m2, e1
	}

	// last resort, decrease m2, lose some digits.
	if toDiv := mu.Pow10(int(ediff)); toDiv > 0 {
		m2 /= toDiv
	} else {
		m2 = 0
	}

	return m1, m2, e1
}

func trailingZeros(value uint64) int {
	var i int
	if value == 0 {
		return 1
	}
	for value%10 == 0 {
		value /= 10
		i++
	}
	return i
}

func jsonMEFormatLen(v Value) int {
	return jsonLen + mu.DecimalDigits(mant(v)) + mu.DecimalLenInt64(int64(exp(v)))
}

func decimalFormatLen(v Value) int {
	if v.IsZero() {
		return 1
	}
	sLen := mu.DecimalDigits(mant(v))
	if e := int(exp(v)); e > 0 { // `exp` trailing zeros
		sLen += e
	} else if e < 0 {
		// TODO(avd) - mantissa may have trailing zeros after the delimeter
		if diff := sLen + e; diff < 0 { // leading zeros
			sLen += -diff
		}
		sLen++ // a delimeter
	}
	return sLen
}

func addWithExp(m1, m2 number, e expType) Value {
	res := m1 + m2
	if res > maxMantissa {
		if e == maxExponent {
			return Max
		}
		res /= 10
		e++
	}
	return combine(res, e)
}

func subWithExp(m1, m2 number, e expType) (v Value, neg bool) {
	var res number
	if m1 >= m2 {
		res = m1 - m2
	} else {
		neg = true
		res = m2 - m1
	}
	return combine(res, e), neg
}

func uint64Cmp(a, b uint64) int {
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

func adjustMantExp(m number, e expType) Value {
	// fix too large matissa, or too small exponent
	for (m > maxMantissa || e < minExponent) && m > 0 && e+1 < maxExponent {
		m /= 10
		e++
	}

	// fix too large exponent
	for e > maxExponent && m*10 < maxMantissa {
		m *= 10
		e--
	}

	if m == 0 || e < minExponent {
		return zero
	}
	if e > maxExponent {
		return Max
	}
	return combine(m, e)
}

func OnlyValue(v Value, _ bool) Value {
	return v
}

func OnlyUint64(u uint64, _ bool) uint64 {
	return u
}
