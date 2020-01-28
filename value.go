// Copyright 2020 Aleksandr Demakin. All rights reserved.

// package fixed implements a fixed-point number, where both mantissa
// and exponent are stored in a single number.
// Can be used to represent currency rates with up to 16 digits of precision.
package fixed

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
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

	decimalFactorTable = []uint64{ // up to 1e18
		1, 10, 100, 1000, 10000,
		100000, 1000000, 10000000, 100000000, 1000000000, 10000000000,
		100000000000, 1000000000000, 10000000000000, 100000000000000,
		1000000000000000, 10000000000000000, 100000000000000000,
		1000000000000000000,
	}

	// 145 zeros, 128 for max exponent, 17 for max mantissa
	manyZeros = "000000000000000000000000000000000000000000000000000000000000" +
		"000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000"

	jsonParts = []string{`{"m":`, `,"e":`, `}`}
	jsonLen   = len(jsonParts[0]) + len(jsonParts[1]) + len(jsonParts[2])

	errRange = fmt.Errorf("value out of range")
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
	maxMantissa = (1<<(bitsInNumber-expBits) - 1)
	maxValue    = maxMantissa | maxExponent

	delim = '.'
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

// Value is a positive fixed-point number.
// 8 high bits are used for exponent, the others are for mantissa.
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
func FromUint64(v uint64) (Value, error) {
	if v > maxMantissa {
		return zero, errRange
	}
	return fromMantAndExp(number(v), 0), nil
}

// FromMantAndExp returns a value for given mantissa and exponent.
func FromMantAndExp(mant uint64, exp int8) (Value, error) {
	if mant > maxMantissa || exp > maxExponent || exp < minExponent {
		return zero, errRange
	}
	return fromMantAndExp(number(mant), exp), nil
}

// FromFloat64 returns a value for given float64.
func FromFloat64(v float64) (Value, error) {
	if v < 0 || math.IsInf(v, 0) || math.IsNaN(v) {
		return zero, fmt.Errorf("bad float number")
	}
	if v == 0 {
		return zero, nil
	}
	m, e := normFloat64(v)
	if e > maxExponent || e == maxExponent && m > maxMantissa {
		return zero, errRange
	}
	if e < minExponent {
		return zero, errRange
	}
	mant, e := decimalMantissa(v, e, 1e-10)
	if e > maxExponent || mant > maxMantissa {
		return maxValue, errRange
	}
	return fromMantAndExp(number(mant), expType(e)).Normalized(), nil
}

// FromString parses a string into a value.
func FromString(s string) (Value, error) {
	s, offset, err := prepareString(s)
	if err != nil {
		return zero, err
	}
	parsed, delimPos, err := parseString(s)
	if err != nil { // could still be a float
		f, fltErr := strconv.ParseFloat(s, 64)
		if fltErr != nil {
			var pe *posError
			if errors.As(err, &pe) {
				pe.pos += offset + 1 // +1 to start indices from 1.
				err = pe
			}
			return zero, fmt.Errorf("parsing failed: %w", err)
		}
		return FromFloat64(f)
	}
	return fromStringAndDelim(parsed, delimPos)
}

// MarshalJSON marshals value according to current JSONMode.
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

// UnmarshalJSON unmarshals a string, float, and an object into a value.
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
		value, err := FromMantAndExp(d.M, d.E)
		if err != nil {
			return err
		}
		*v = value
	default:
		value, err := FromString(string(data))
		if err != nil {
			return err
		}
		*v = value
	}
	return nil
}

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
// returns a string, where
//	- all leading zeros before a delimiter are omitted.
//	- all trailing zeros after a delimeter are omitted.
// also returns a delimeter's position, or -1 if if does not present in the string.
func parseString(s string) (result string, delimPos int, err error) {
	var b strings.Builder
	delimPos = -1
	for i, r := range s {
		switch {
		case '0' <= r && r <= '9':
			if r == '0' && b.Len() == 0 && delimPos == -1 { // omit leading zeros
				continue
			}
			b.WriteRune(r)
		case r == delim:
			if delimPos >= 0 {
				return "", -1, newPosError("unexpected delimeter", i)
			}
			delimPos = b.Len()
		default:
			return "", -1, newPosError(fmt.Sprintf("unexpected symbol %q", r), i)
		}
	}
	result = b.String()
	if delimPos >= 0 {
		for { // remove trailing zeros after the delimeter.
			l := len(result)
			if l < 2 || result[l-1] != '0' || l <= delimPos {
				break
			}
			result = result[:len(result)-1]
		}
	}
	return result, delimPos, nil
}

// fromStringAndDelim parses a string into Value.
// accepts a string without leading zeros before a delimeter and trailing zeros after the delimeter.
func fromStringAndDelim(s string, delimPos int) (Value, error) {
	e := int8(0)
	if delimPos >= 0 {
		diff := delimPos - len(s)
		if diff < minExponent {
			return zero, errRange
		}
		e = int8(diff)
	}
	if trimmed := strings.TrimRight(s, "0"); len(trimmed) < len(s) {
		diff := len(s) - len(trimmed)
		if int(e)+diff > maxExponent {
			return zero, errRange
		}
		e += int8(diff)
		s = trimmed
	}
	s = strings.TrimLeft(s, "0")
	if len(s) == 0 {
		return zero, nil
	}
	u, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return zero, err
	}
	if u > maxMantissa {
		return zero, errRange
	}
	return fromMantAndExp(number(u), e), nil
}

// MantUint64 returns v's mantissa as is.
func (v Value) MantUint64() uint64 {
	return uint64(mant(v))
}

// ToExp changes the mantissa of v so, that v = m * 10e'exp',
func (v Value) ToExp(exp int) Value {
	if exp > maxExponent {
		return maxValue
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

// Uint64 returns integer part of the value.
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
		return maxValue, false
	}
	if e < 0 {
		return m / p, trailingZeros(m) >= int(-e)
	}
	intPart, frac := maxMantissa/m, maxMantissa%m
	e2 := uint64Len(intPart) - 1
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

// String returns string representation of the value.
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

// Normalized eliminates trailing zeros in the fractional part.
func (v Value) Normalized() Value {
	v.normalize()
	return v
}

func (v *Value) normalize() {
	m, e := split(*v)
	if m == 0 {
		*v = zero
		return
	}
	// remove trailing zeros
	for m%10 == 0 && e < maxExponent {
		m /= 10
		e++
	}
	*v = fromMantAndExp(number(m), e)
}

func abs(val int) int {
	if val < 0 {
		return -val
	}
	return val
}

func pow10(pow int) uint64 {
	if pow < 0 || pow >= len(decimalFactorTable) {
		return 0
	}
	return decimalFactorTable[pow]
}

func int64Len(value int64) int {
	result := 0
	if value < 0 {
		result++
		value = -value
	}
	return result + uint64Len(uint64(value))
}

func uint64Len(value uint64) int {
	result := 1
	for value > 9 {
		value /= 10
		result++
	}
	return result
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
	return jsonLen + uint64Len(mant(v)) + int64Len(int64(exp(v)))
}

func calcStrLen(v Value) int {
	v = v.Normalized()
	mantLen := uint64Len(mant(v))
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

// decimalMantissa receives a number 'f' and an exponent 'e', such as in the range 1 <= f*(10**e) <= 10
// it returns decimal mantissa and e, so that mant*(10**-e) ~= f
func decimalMantissa(f float64, e int, epsilon float64) (mant uint64, exp int) {
	const maxPrec = 16
	var result uint64
	for i := 0; i < maxPrec; i++ {
		integ, frac := math.Modf(f * math.Pow10(e+i))
		result = uint64(integ)
		if frac < epsilon {
			return result, -(e + i)
		}
	}
	return result, -(e + maxPrec)
}
