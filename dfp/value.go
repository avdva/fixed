// Copyright 2020 Aleksandr Demakin. All rights reserved.

// Package dfp implements a decimal floating-point number, where both mantissa
// and exponent are stored in a single number.
// Can be used to represent currency rates with up to 16 digits of precision.
package dfp

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"strconv"
	"strings"
	"unicode"
	"unsafe"
)

var (
	// JSONMode defines the way all values are marshaled into json, see JSONMode* constants.
	// This variable is not thread-safe, so this should be changed on program start.
	JSONMode = JSONModeCompact
)

const (
	// JSONModeString produces values as strings, like `"1234.5678"`
	JSONModeString = iota
	// JSONModeFloat marshals values as floats, like `1234.5678`.
	JSONModeFloat
	// JSONModeME marshals values with mantissa and exponent, like `{"m":123,"e":-5}`.
	JSONModeME
	// JSONModeCompact will choose the shortest form between JSONModeString and JSONModeME.
	JSONModeCompact
)

var (
	zero Value

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

	// 145 zeros, 128 for max exponent, 17 for max mantissa
	manyZeros = "000000000000000000000000000000000000000000000000000000000000" +
		"000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000"

	jsonParts = []string{`{"m":`, `,"e":`, `}`}
	jsonLen   = len(jsonParts[0]) + len(jsonParts[1]) + len(jsonParts[2])

	digitsInMaxMantissa = decimalDigits(maxMantissa)
)

const (
	bitsInNumber = unsafe.Sizeof(number(0)) * 8
	expBits      = 8
	mantBits     = bitsInNumber - expBits

	expMask    = 1<<expBits - 1
	highestBit = 1 << (bitsInNumber - 1)
	mantMask   = (highestBit - 1 | highestBit) >> expBits

	maxExponent = (1<<(expBits-1) - 1)
	minExponent = -maxExponent
	// maxMantissa is 72057594037927935 for a (8,56) number
	maxMantissa = (1<<(bitsInNumber-expBits) - 1)
	minMantissa = 1
	maxNumber   = 1<<bitsInNumber - 1

	delim = '.'
)

const (
	// Max is the maximum possible value.
	Max = Value(number(maxExponent)<<mantBits | (maxMantissa & mantMask))
	// Min is the minimum possible value.
	Min = Value(number((1<<(expBits-1)+1))<<mantBits | minMantissa)
)

type posError struct {
	pos int
	err string
}

func newPosError(err string, pos int) *posError {
	return &posError{err: err, pos: pos}
}

func (pe posError) Error() string {
	return pe.err + fmt.Sprintf(" at pos %d", pe.pos)
}

type (
	number  = uint64
	expType = int8
)

// Value is a positive floating-point number.
// It currently uses a uint64 value as a data type, where
// 8 bits are used for exponent and 56 for mantissa.
//   63      55                                                     0
//   ________|_______________________________________________________
//   eeeeeeeemmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmm
//
// Negative numbers aren't currently supported.
// Value can be useful for representing numbers like prices in financial services.
type Value number

func exp(v Value) expType {
	return expType(v >> mantBits & expMask)
}

func mant(v Value) number {
	return number(v & mantMask)
}

func split(v Value) (mantissa number, exponent expType) {
	return mant(v), exp(v)
}

func fromMantAndExp(mant number, exp expType) Value {
	return Value(number(exp)<<mantBits | (mant & mantMask))
}

// FromUint64 returns a value for given uint64 number.
// If the number cannot be precisely represented, the least significant digits will be truncated.
func FromUint64(v uint64) Value {
	return adjustMantExp(number(v), 0)
}

// FromMantAndExp returns a value for given mantissa and exponent.
// If the number cannot be precisely represented, the least significant digits will be truncated.
func FromMantAndExp(mant uint64, exp int8) Value {
	return adjustMantExp(number(mant), int(exp))
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
	_, e := normFloat64(v)
	mant, e := decimalMantissa(v, e, 1e-10)
	return adjustMantExp(number(mant), e).Normalized(), nil
}

// FromString parses a string into a value.
func FromString(s string) (Value, error) {
	s, offset, err := prepareString(s)
	if err != nil {
		return zero, err
	}
	parsed, delimPos, err := parseString(s)
	if err != nil { // could still be a float
		if f, fltErr := strconv.ParseFloat(s, 64); fltErr == nil {
			return FromFloat64(f)
		}
		var pe *posError
		if errors.As(err, &pe) { // try to locate error position.
			pe.pos += offset + 1 // add what we've trimmed before and +1 to start indices from 1.
			err = pe
		}
		return zero, fmt.Errorf("parsing failed: %w", err)
	}
	return fromStringAndDelim(parsed, delimPos), nil
}

// MarshalJSON marshals value according to current JSONMode.
// See JSONMode and JSONMode* constants.
func (v Value) MarshalJSON() ([]byte, error) {
	return v.toJSON(JSONMode), nil
}

func (v Value) toJSON(mode int) []byte {
	switch mode {
	case JSONModeFloat:
		return []byte(strconv.FormatFloat(v.Float64(), 'f', -1, 64))
	case JSONModeME:
		var builder strings.Builder
		v = v.Normalized()
		builder.WriteString(jsonParts[0])
		builder.WriteString(strconv.FormatUint(mant(v), 10))
		builder.WriteString(jsonParts[1])
		builder.WriteString(strconv.FormatInt(int64(exp(v)), 10))
		builder.WriteString(jsonParts[2])
		return []byte(builder.String())
	case JSONModeCompact:
		if calcStrLen(v) <= calcMeLen(v) {
			return v.toJSON(JSONModeString)
		}
		return v.toJSON(JSONModeME)
	default: // marshal as a string
		var builder strings.Builder
		builder.WriteRune('"')
		v.toStringsBuilder(&builder)
		builder.WriteRune('"')
		return []byte(builder.String())
	}
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

// prepareString cleans the string from ",-,+ symbols, and spaces.
func prepareString(s string) (prepared string, offset int, err error) {
	if len(s) == 0 {
		return "", 0, fmt.Errorf("empty input")
	}
	if s[0] == '"' {
		s = s[1:]
		offset++
	}
	if len(s) == 0 {
		return "", 0, fmt.Errorf("empty input")
	}
	if s[len(s)-1] == '"' {
		s = s[:len(s)-1]
	}
	if trimmed := strings.TrimLeftFunc(s, unicode.IsSpace); len(trimmed) != len(s) {
		offset += len(s) - len(trimmed)
		s = trimmed
	}
	s = strings.TrimRightFunc(s, unicode.IsSpace)
	if len(s) == 0 {
		return "", 0, fmt.Errorf("empty input")
	}
	if s[0] == '-' {
		return "", 0, fmt.Errorf("negative value")
	}
	if s[0] == '+' {
		offset++
		s = s[1:]
	}
	return s, offset, nil
}

// parseString checks if the string can be converted into a Value.
// returns a string without leading and trailing zeros, and an exponent
func parseString(s string) (result string, e int, err error) {
	result, delimPos, err := removeLeadingZeros(s)
	if err != nil {
		return "", 0, err
	}
	result, e = removeTrailingZerosString(result, delimPos)
	return result, e, nil
}

func removeLeadingZeros(s string) (result string, delimPos int, err error) {
	var b strings.Builder
	delimPos, firstNonZeroPos := -1, -1
	for i, r := range s {
		switch {
		case '0' <= r && r <= '9':
			if b.Len() == 0 {
				if r == '0' { // trim leading zeros
					continue
				}
				firstNonZeroPos = i
			}
			b.WriteRune(r)
		case r == delim:
			if delimPos != -1 {
				return "", 0, newPosError("unexpected delimeter", i)
			}
			delimPos = i
		default:
			return "", 0, newPosError(fmt.Sprintf("unexpected symbol %q", r), i)
		}
	}
	if firstNonZeroPos == -1 { // a zero-only string
		return "", 0, nil
	}

	result = b.String()

	// move delimPos to the beginning of the trimmed string
	if delimPos >= 0 {
		if delimPos < firstNonZeroPos {
			firstNonZeroPos--
		}
		delimPos -= firstNonZeroPos
	} else { // if there is no delim, add one at the end of the string 123 --> 123.
		delimPos = len(result)
	}

	return result, delimPos, nil
}

func removeTrailingZerosString(s string, delimPos int) (result string, e int) {
	for {
		l := len(s)
		if l == 0 || s[l-1] != '0' {
			break
		}
		s = s[:l-1]
	}
	return s, delimPos - len(s)
}

// fromStringAndDelim parses a string without leading and trailing zeros into Value.
func fromStringAndDelim(s string, e int) Value {
	if len(s) == 0 {
		return zero
	}
	if toCut := len(s) - digitsInMaxMantissa; toCut > 0 {
		expInc := toCut
		if e < 0 {
			expInc -= -e
		}
		if expInc > 0 {
			e += expInc
		}
		s = s[:digitsInMaxMantissa]
	}
	u, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		panic(err) // should not normally happen
	}
	return adjustMantExp(number(u), e)
}

// MantUint64 returns v's mantissa as is.
func (v Value) MantUint64() uint64 {
	return uint64(mant(v))
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

// Cmp compares two values.
// Returns -1 if a < b, 0 if a == b, 1 if a > b
func (v Value) Cmp(other Value) int {
	m1, e1 := split(v)
	m2, e2 := split(other)
	ediff := int(e1 - e2)
	if ediff == 0 || m1 == 0 || m2 == 0 {
		return uint64Cmp(m1, m2)
	}
	maxDigit1 := int(e1) + decimalDigits(m1)
	maxDigit2 := int(e2) + decimalDigits(m2)
	if maxDigit1 > maxDigit2 {
		return 1
	} else if maxDigit1 < maxDigit2 {
		return -1
	}
	if ediff > 0 {
		m1 *= pow10(ediff)
	} else {
		m2 *= pow10(-ediff)
	}
	return uint64Cmp(m1, m2)
}

// Add sums two values.
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

// sub returns |a-b|.
func (v Value) sub(other Value) Value {
	m1, e1 := split(v)
	m2, e2 := split(other)
	// first, check for obvious cases, when one of the arguments is zero
	if m2 == 0 {
		if m1 == 0 {
			return zero
		}
		return v
	}
	if m1 == 0 {
		return other
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
	e := int(e1) + int(e2)
	// perform a 128-bit multiplication
	hi, lo := bits.Mul64(uint64(m1), uint64(m2))
	if hi > 0 {
		// the result overflows uint64, so we'll divide it by a factor of 10,
		// so that it fits a uint64 value again, and add that factor to the resulting exponent.
		powDiv := decimalDigits(hi) + 1
		toDiv := pow10(powDiv)
		e += int(powDiv)
		lo, _ = bits.Div64(hi, lo, toDiv)
	}

	return adjustMantExp(lo, e)
}

// DivMod calculates such quo and rem, that a = b * quo + rem. If b == 0, Div panics.
// Quo will be rounded to prec digits.
// Notice that prec can be negative.
func (v Value) DivMod(other Value, prec int) (quo, rem Value) {
	v, other = v.Normalized(), other.Normalized()
	q, r, e := div(v, other)
	if r == 0 {
		return adjustMantExp(q, e).Normalized(), zero
	}
	if e < maxExponent {
		var et expType
		q, et = trimZeros(q, expType(e), maxExponent)
		e = int(et)
	}
	q, e = roundToPrec(q, e, prec)
	quo = adjustMantExp(q, e)
	return quo, v.sub(other.Mul(quo))
}

func roundToPrec(n number, e, prec int) (number, int) {
	var toCut int
	dd := decimalDigits(n)
	if prec >= 0 {
		toCut = -e - prec
	} else {
		if e < 0 {
			dd += e
			if dd <= 0 { // no digits before the decimal point
				return 0, 0
			}
			n /= pow10(-e)
			e = 0
		}
		toCut = -prec
	}
	if toCut > 0 {
		if toCut > dd {
			toCut = dd
		}
		n /= pow10(toCut)
		e += toCut
	}
	return n, e
}

// Div calculates a/b. If b == 0, Div panics.
// First, it tries to perform integer division, and if the remainder is zero, returns the result.
// Otherwise, it returns the result of a float64 division.
func (v Value) Div(other Value) Value {

	v, other = v.Normalized(), other.Normalized()

	if quo, rem, e := div(v, other); rem == 0 {
		return adjustMantExp(quo, e).Normalized()
	}

	return float64Div(v, other)
}

func float64Div(a, b Value) Value {
	m1, e1 := split(a)
	m2, e2 := split(b)
	if m2 == 0 {
		return zero
	}
	flt := float64(m1) / float64(m2) * math.Pow10(int(e1)-int(e2))
	result, _ := FromFloat64(flt)
	return result
}

func div(v1, v2 Value) (quo, rem number, e int) {

	m1, e1 := split(v1)
	m2, e2 := split(v2)

	if m2 == 0 {
		panic("division by zero")
	}
	if m1 == 0 {
		return 0, 0, 0
	}

	// a*10^e1 / b*10^e2 = (a/b) * 10^(e1-e2)
	e = int(e1) - int(e2)

	if m1%m2 == 0 {
		return m1 / m2, 0, e
	}

	// give it best chances to division.
	// shift m1 close to the maximum possible number.
	toMult := log10(maxNumber / m1)
	m1 *= pow10(toMult)
	e -= toMult

	return m1 / m2, m1 % m2, e
}

// toEqualExp changes m1 and m2 in such a way, that e1 == e2.
// the result can be used to calculate m1+m2, m1-m2.
// if the difference between the exponents is too big, m2 can lose some (or all) digits.
func toEqualExp(m1 number, e1 expType, m2 number, e2 expType) (number, number, expType) {

	if e1 >= e2 {
		return doToEqualExp(m1, e1, m2, e2)
	}

	r1, r2, re := doToEqualExp(m2, e2, m1, e1)
	return r2, r1, re
}

// doToEqualExp is a helper for toEqualExp. it assumes that e1 >= e2.
func doToEqualExp(m1 number, e1 expType, m2 number, e2 expType) (number, number, expType) {
	ediff := int(e1) - int(e2)
	if ediff == 0 {
		return m1, m2, e1
	}

	// try to trim trailing zeros for m2.
	// if we have enough to increase e2 so that it equals e1, return the sum.
	m2, e2 = trimZeros(m2, e2, e1)
	if ediff = int(e1) - int(e2); ediff == 0 {
		return m1, m2, e1
	}

	// next, try to increase m1 and decrease e1 so, that e1 == e2.
	toMult := log10(maxMantissa / m1)
	if toMult > ediff {
		toMult = ediff
	}
	m1 *= pow10(toMult)
	e1 -= int8(toMult)

	if ediff = int(e1) - int(e2); ediff == 0 {
		return m1, m2, e1
	}

	// last resort, decrease m2, lose some digits.
	if toDiv := pow10(ediff); toDiv > 0 {
		m2 /= toDiv
	} else {
		m2 = 0
	}

	return m1, m2, e1
}

func log10(a uint64) int {
	return decimalDigits(a) - 1
}

// ToExp changes the mantissa of v so, that v = m * 10e'exp'.
// As a result, mantissa can lose some digits in precision, become zero, or Max.
func (v Value) ToExp(exp int) Value {
	if exp > maxExponent {
		return Max
	}
	if exp < minExponent {
		return fromMantAndExp(0, minExponent)
	}
	m, e := split(v)
	diff := int(e) - exp
	if diff == 0 {
		return v
	}
	p := pow10(abs(diff))
	if p == 0 {
		return fromMantAndExp(0, expType(exp))
	}
	if diff > 0 {
		if maxMantissa/(m) < p {
			m = maxMantissa
		} else {
			m *= p
		}
	} else {
		m /= p
	}
	return fromMantAndExp(m, expType(exp))
}

// Uint64 returns the value as a uint64 number.
func (v Value) Uint64() uint64 {
	value, _ := v.toUint64()
	return value
}

func (v Value) toUint64() (value uint64, exact bool) {
	v = v.Normalized()
	e, m := exp(v), mant(v)
	if m == 0 {
		return 0, true
	}
	if e == 0 {
		return m, true
	}
	p := pow10(abs(int(e)))
	if p == 0 {
		return maxMantissa, false
	}
	if e < 0 {
		return m / p, trailingZeros(m) >= int(-e)
	}
	intPart, frac := maxMantissa/m, maxMantissa%m
	e2 := decimalDigits(intPart) - 1
	if e2 > int(e) || e2 == int(e) && maxMantissa-frac <= intPart*m {
		return m * p, true
	}
	return maxMantissa, false
}

// Float64 returns a float64 value.
func (v Value) Float64() float64 {
	return float64(mant(v)) * math.Pow10(int(exp(v)))
}

// GoString returns debug string representation.
func (v Value) GoString() string {
	m, e := split(v)
	return v.String() + fmt.Sprintf(" {%v, %v}", m, e)
}

// String returns a string representation of the value.
func (v Value) String() string {
	if mant(v) == 0 {
		return "0"
	}
	var builder strings.Builder
	v.toStringsBuilder(&builder)
	return builder.String()
}

func (v Value) toStringsBuilder(builder *strings.Builder) {
	v = v.Normalized()
	m, e := split(v)
	s := strconv.FormatUint(m, 10)
	switch {
	case e == 0:
		builder.WriteString(s)
	case e > 0:
		builder.WriteString(s)
		builder.WriteString(manyZeros[:int(e)])
	default:
		if diff := len(s) + int(e); diff <= 0 { // add leading zeros and a delimiter
			builder.WriteRune('0')
			builder.WriteRune(delim)
			builder.WriteString(manyZeros[:-diff])
			builder.WriteString(s)
		} else { // insert a delimeter
			builder.WriteString(s[:diff])
			builder.WriteRune(delim)
			builder.WriteString(s[diff:])
		}
	}
}

// Normalized eliminates trailing zeros in the mantissa.
// The process increases the exponent, and stops if it exceeds the maximum possible exponent,
// so that it is possible, that the mantissa will still have trailing zeros.
func (v Value) Normalized() Value {
	m, e := split(v)
	if m == 0 {
		return zero
	}
	return fromMantAndExp(trimZeros(m, e, maxExponent))
}

func abs(val int) int {
	mask := val >> (unsafe.Sizeof(0)*8 - 1)
	return (val + mask) ^ mask
}

func pow10(pow int) uint64 {
	if pow < 0 || pow >= len(decimalFactorTable) {
		return 0
	}
	return decimalFactorTable[pow]
}

func int64DecimalLen(value int64) int {
	result := 0
	if value < 0 {
		result++
		value = -value
	}
	return result + decimalDigits(uint64(value))
}

func binaryDigits(value uint64) int {
	return int(8*unsafe.Sizeof(uint64(0))) - bits.LeadingZeros64(value)
}

// decimalDigits returns the number of decimal digits in 'value'.
// see https://stackoverflow.com/a/25934909
func decimalDigits(value uint64) int {
	if value == 0 {
		return 1
	}

	digits := digitsHelper[binaryDigits(value)]
	if value >= decimalFactorTable[digits] {
		digits++
	}
	return digits
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

func calcMeLen(v Value) int {
	return jsonLen + decimalDigits(mant(v)) + int64DecimalLen(int64(exp(v)))
}

func calcStrLen(v Value) int {
	v = v.Normalized()
	mantLen := decimalDigits(mant(v))
	// the length of the string. 2 for a pair of quotes plus len of mantissa
	sLen := 2 + mantLen
	if e := int(exp(v)); e > 0 { // `exp` trailing zeros
		sLen += e
	} else {
		sLen++                             // a delimeter
		if diff := e + mantLen; diff < 0 { // leading zeros
			sLen += -diff
		}
	}
	return sLen
}

// normFloat64 calculates such e, that 1 <= f*(10**e) <= 10
// returns f*(10**e), e
func normFloat64(f float64) (pow float64, exp int) {
	if f <= 0 || math.IsInf(f, 0) || math.IsNaN(f) {
		return 0, 0
	}
	var e int
	switch {
	case f < 1:
		e = int(math.Log10(1/f)) + 1
	case f > 10:
		e = -(int(math.Log10(f/10)) + 1)
	default:
		return f, 0
	}
	return f * math.Pow10(e), e
}

// decimalMantissa receives a number 'f' and an exponent 'e', such as 1 <= f*(10**e) <= 10
// it returns integer mantissa and e, so that abs(mant*(10^-e) - f) < epsilon.
func decimalMantissa(f float64, e int, epsilon float64) (mant uint64, exp int) {
	const maxPrec = 16
	var result uint64
	var i int
	for ; ; i++ {
		integ, frac := math.Modf(f * math.Pow10(e+i))
		result = uint64(integ)
		if frac < epsilon || i >= maxPrec {
			break
		}
	}
	return result, -(e + i)
}

func trimZeros(m number, e, eMax expType) (number, expType) {
	for e < eMax && m%10 == 0 {
		m /= 10
		e++
	}
	return m, e
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
	return fromMantAndExp(res, e)
}

func subWithExp(m1, m2 number, e expType) Value {
	var res number
	if m1 >= m2 {
		res = m1 - m2
	} else {
		res = m2 - m1
	}
	return fromMantAndExp(res, e)
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

func adjustMantExp(m number, e int) Value {
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
	return fromMantAndExp(m, expType(e))
}
