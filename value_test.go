// Copyright 2020 Aleksandr Demakin. All rights reserved.

package fixed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNormFloat64(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		f   float64
		res float64
		e   int
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
			f, e := normFloat64(test.f)
			a.InDelta(test.res, f, 1e10)
			a.Equal(test.e, e)
		})
	}
}

func TestFromFloat(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		f   float64
		v   Value
		err string
	}{
		{0, zero, ""},
		{0.012345, fromMantAndExp(12345, -6), ""},
		{123450000, fromMantAndExp(12345, 4), ""},
		{maxMantissa / 100, fromMantAndExp(maxMantissa/100, 0), ""},
		{0.12345e-50, fromMantAndExp(12345, -55), ""},
		{math.Pow10(maxExponent), fromMantAndExp(1, maxExponent), ""},
		{math.Pow10(minExponent), fromMantAndExp(1, minExponent), ""},
		{math.Pow10(maxExponent + 1), fromMantAndExp(10, maxExponent), ""},
		{math.Pow10(minExponent - 1), zero, ""},
		{float64(15) / 7, fromMantAndExp(21428571428571428, -16), ""},

		{-1, zero, "bad float number"},
		{math.Inf(1), zero, "bad float number"},
		{math.Inf(-1), zero, "bad float number"},
		{math.NaN(), zero, "bad float number"},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			v, err := FromFloat64(test.f)
			if len(test.err) == 0 {
				if a.NoError(err) {
					a.Equal(test.v, v)
					asFloat := test.v.Float64()
					if asFloat == 0 || test.f == 0 {
						a.InDelta(0, asFloat, 1e-15)
					} else {
						a.InEpsilon(test.f, asFloat, 1e-15)
					}
				}
			} else {
				a.EqualError(err, test.err)
			}
		})
	}
}

func TestFromString(t *testing.T) {
	a := assert.New(t)
	testNum := uint64(1234567890121416182)
	testNumStr := strconv.FormatUint(testNum, 10)
	dd, e := cutToDigits(testNum, digitsInMaxMantissa)
	tests := []struct {
		s string
		v Value
		e string
	}{
		{" 0 ", zero, ""},
		{" 00000.00000", zero, ""},
		{"+.00000   ", zero, ""},
		{`"+.00000   "`, zero, ""},
		{"  .00000", zero, ""},
		{`"  .00000"`, zero, ""},
		{"00001.10000", fromMantAndExp(11, -1), ""},
		{"+000010.01000", fromMantAndExp(1001, -2), ""},
		{manyZeros + strconv.FormatUint(maxMantissa, 10) + "." + manyZeros, fromMantAndExp(maxMantissa, 0), ""},
		{strconv.FormatUint(maxMantissa, 10) + "." + manyZeros, fromMantAndExp(maxMantissa, 0), ""},
		{"0.01234500", fromMantAndExp(12345, -6), ""},
		{"123450000", fromMantAndExp(12345, 4), ""},
		{"123450000.", fromMantAndExp(12345, 4), ""},
		{"0123450000.", fromMantAndExp(12345, 4), ""},
		{"0123450000.0", fromMantAndExp(12345, 4), ""},
		{".123450000", fromMantAndExp(12345, -5), ""},
		{"0.123450000", fromMantAndExp(12345, -5), ""},
		{"012.3450", fromMantAndExp(12345, -3), ""},
		{"12.345", fromMantAndExp(12345, -3), ""},
		{strconv.FormatUint(maxMantissa, 10), fromMantAndExp(maxMantissa, 0), ""},
		{strconv.FormatUint(maxMantissa/100, 10), fromMantAndExp(maxMantissa/100, 0), ""},
		{"0.12345e-50", fromMantAndExp(12345, -55), ""},
		{"1e" + strconv.Itoa(maxExponent+1), fromMantAndExp(10, maxExponent), ""},
		{"1e" + strconv.FormatInt(minExponent-1, 10), zero, ""},
		{"123e" + strconv.Itoa(maxExponent+20), Max, ""},
		{testNumStr + "0", fromMantAndExp(dd, expType(1+e)), ""},
		{"000" + testNumStr + "00000", fromMantAndExp(dd, expType(5+e)), ""},
		{"000" + testNumStr + "00000.00000000", fromMantAndExp(dd, expType(5+e)), ""},
		{testNumStr + debugZeroStr(maxExponent-10), fromMantAndExp(dd, maxExponent-10+expType(e)), ""},
		{"." + debugZeroStr(maxExponent-10) + testNumStr, fromMantAndExp(dd/pow10(9), minExponent), ""},
		{"", zero, "empty input"},
		{`"`, zero, "empty input"},
		{`  ""  `, zero, "parsing failed: unexpected symbol '\"' at pos 3"},
		{`"   -"`, zero, "negative value"},
		{`abc`, zero, "parsing failed: unexpected symbol 'a' at pos 1"},
		{`  "abc`, zero, "parsing failed: unexpected symbol '\"' at pos 3"},
		{`   0.00.  `, zero, "parsing failed: unexpected delimeter at pos 8"},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			v, err := FromString(test.s)
			if len(test.e) == 0 {
				if a.NoError(err) {
					a.Equal(test.v, v, test.s)
				}
			} else {
				a.EqualError(err, test.e)
			}
		})
	}
}

func debugZeroStr(count int) string {
	var b bytes.Buffer
	for i := 0; i < count/len(manyZeros); i++ {
		b.WriteString(manyZeros)
	}
	if rem := count % len(manyZeros); rem > 0 {
		b.WriteString(manyZeros[:rem])
	}
	return b.String()
}

func TestFromMantAndExp(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		m, mExpected number
		e            expType
	}{
		{0, 0, 0},
		{123456, 123456, 0},
		{123456, 123456, 35},
		{maxMantissa, maxMantissa, maxExponent},
		{maxMantissa, maxMantissa, minExponent},
		{maxMantissa, maxMantissa, 1},
		{maxMantissa, maxMantissa, -1},
		{maxMantissa + 1, 0, minExponent},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			v := fromMantAndExp(test.m, test.e)
			m, e := split(v)
			a.Equal(test.e, e)
			a.Equal(test.mExpected, m)
		})
	}
}

func TestUint64(t *testing.T) {
	type testItem struct {
		expected uint64
		exact    bool
		v        Value
	}
	a := assert.New(t)
	tests := []testItem{
		{123456, true, fromMantAndExp(123456, 0)},
		{maxMantissa, true, fromMantAndExp(maxMantissa, 0)},
		{1234, false, fromMantAndExp(123456, -2)},
		{0, true, fromMantAndExp(0, 0)},
		{0, true, fromMantAndExp(0, 127)},
		{1, true, fromMantAndExp(1e16, -16)},

		{12, true, fromMantAndExp(12000, -3)},
		{maxMantissa / uint64(1e10), false, fromMantAndExp(maxMantissa, -10)},
		{1, true, fromMantAndExp(1000, -3)},

		{maxMantissa, false, fromMantAndExp(maxMantissa, 1)},
		{maxMantissa, false, fromMantAndExp(maxMantissa-1, 1)},
		{maxMantissa, false, fromMantAndExp(maxMantissa/10+1, 1)},
		{(maxMantissa / 100) * 10, true, fromMantAndExp(maxMantissa/100, 1)},

		{(maxMantissa / 10) * 10, true, fromMantAndExp(maxMantissa/10, 1)},
	}
	for i, item := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			u, exact := item.v.toUint64()
			a.Equal(item.expected, u)
			a.Equal(item.exact, exact)

			if item.exact {
				a.Equal(item.v.Normalized(), FromUint64(item.expected).Normalized())
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	type testItem struct {
		expected, actual Value
	}
	a := assert.New(t)
	tests := []testItem{
		{zero, fromMantAndExp(0, 0)},
		{zero, fromMantAndExp(0, 3)},
		{fromMantAndExp(123, 2), fromMantAndExp(12300, 0)},
		{fromMantAndExp(123, 3), fromMantAndExp(12300, 1)},
		{fromMantAndExp(123, 0), fromMantAndExp(123, 0)},
		{fromMantAndExp(123, 0), fromMantAndExp(123000000000, -9)},
		{fromMantAndExp(123, 5), fromMantAndExp(123000000000, -4)},
		{fromMantAndExp(123456789, 0), fromMantAndExp(123456789, 0)},
		{fromMantAndExp(123456789, 4), fromMantAndExp(123456789, 4)},
		{fromMantAndExp(12345, minExponent+4), fromMantAndExp(123450000, minExponent)},

		{fromMantAndExp(12345, maxExponent), fromMantAndExp(123450, maxExponent-1)},
		{fromMantAndExp(123450, maxExponent), fromMantAndExp(1234500, maxExponent-1)},
		{fromMantAndExp(1234500, maxExponent), fromMantAndExp(12345000, maxExponent-1)},
		{fromMantAndExp(1, minExponent+16), fromMantAndExp(1e16, minExponent)},
	}
	for i, item := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(item.expected, item.actual.Normalized())
		})
	}
}

func TestToEqualExp(t *testing.T) {
	type testItem struct {
		m1     number
		e1     expType
		m2     number
		e2     expType
		r1, r2 number
		re     expType
	}
	a := assert.New(t)
	tests := []testItem{
		{1, 0, 1, 0, 1, 1, 0},
		{maxMantissa, 0, maxMantissa, 0, maxMantissa, maxMantissa, 0},
		{1234, 5, 123450, 4, 1234, 12345, 5},
		{12345600000, 5, 98765, 10, 123456, 98765, 10},
		{1234, 10, 12345, 4, 1234000000, 12345, 4},
		{maxMantissa / 1000, 3, 1, 0, (maxMantissa / 1000) * 1000, 1, 0},
		{123, minExponent + 2, 456788, minExponent, 12300, 456788, minExponent},
		{maxMantissa, maxExponent, maxMantissa, maxExponent - 10, maxMantissa, maxMantissa / uint64(1e10), maxExponent},
		{
			maxMantissa,
			maxExponent,
			maxMantissa,
			maxExponent - expType(digitsInMaxMantissa) + 1,
			maxMantissa,
			maxMantissa / pow10(digitsInMaxMantissa-1),
			maxExponent,
		},
		{maxMantissa, maxExponent, maxMantissa, maxExponent - expType(digitsInMaxMantissa), maxMantissa, 0, maxExponent},
		{
			maxMantissa,
			minExponent + expType(digitsInMaxMantissa) - 1,
			maxMantissa,
			minExponent,
			maxMantissa,
			maxMantissa / (pow10(digitsInMaxMantissa - 1)),
			minExponent + +expType(digitsInMaxMantissa) - 1,
		},
		{
			maxMantissa,
			minExponent + expType(digitsInMaxMantissa),
			maxMantissa,
			minExponent,
			maxMantissa,
			0,
			minExponent + +expType(digitsInMaxMantissa),
		},
		{
			(maxMantissa / 1000) * 1000,
			minExponent + expType(digitsInMaxMantissa) - 2,
			maxMantissa,
			minExponent,
			(maxMantissa / 1000) * 1000,
			maxMantissa / pow10(digitsInMaxMantissa-2),
			minExponent + expType(digitsInMaxMantissa) - 2,
		},
		{maxMantissa, minExponent + 12, 1e15, minExponent, maxMantissa, 1000, minExponent + 12},
		{
			maxMantissa / 1000,
			minExponent + 12,
			maxMantissa,
			minExponent,
			(maxMantissa / 1000) * 1000,
			maxMantissa / uint64(1e9),
			minExponent + 9,
		},
	}
	for i, item := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			r1, r2, re := toEqualExp(item.m1, item.e1, item.m2, item.e2)
			a.Equal(item.r1, r1)
			a.Equal(item.r2, r2)
			a.Equal(item.re, re)
			r1, r2, re = toEqualExp(item.m2, item.e2, item.m1, item.e1)
			a.Equal(item.r1, r2)
			a.Equal(item.r2, r1)
			a.Equal(item.re, re)
		})
	}
}

func TestFloat64(t *testing.T) {
	type testItem struct {
		expected, actual float64
	}
	a := assert.New(t)
	tests := []testItem{
		{0, fromMantAndExp(0, 0).Float64()},
		{123456, fromMantAndExp(123456, 0).Float64()},
		{0.123456, fromMantAndExp(123456, -6).Float64()},
		{456.789, fromMantAndExp(456789, -3).Float64()},
		{123 * math.Pow10(maxExponent), fromMantAndExp(123, maxExponent).Float64()},
		{0, fromMantAndExp(0, 3).Float64()},
		{0, fromMantAndExp(0, 0).Float64()},
		{float64(maxMantissa) * math.Pow10(maxExponent), fromMantAndExp(maxMantissa, maxExponent).Float64()},
		{float64(maxMantissa) * math.Pow10(minExponent), fromMantAndExp(maxMantissa, minExponent).Float64()},
	}
	for i, item := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(item.expected, item.actual)
		})
	}
}

func TestString(t *testing.T) {
	type testItem struct {
		expected, actual string
	}
	a := assert.New(t)
	maxMantissaStr := strconv.FormatUint(maxMantissa, 10)
	addE := func(zeros int, neg bool) string {
		l := len(zeroStr(zeros))
		if l == zeros {
			return ""
		}
		result := "e"
		if neg {
			result += "-"
		}
		return result + strconv.Itoa(zeros-l)
	}
	tests := []testItem{
		{"123456", fromMantAndExp(123456, 0).String()},
		{"0.123456", fromMantAndExp(123456, -6).String()},
		{"0.0000123456", fromMantAndExp(123456, -10).String()},
		{"0.00123456", fromMantAndExp(123456, -8).String()},
		{"123.456", fromMantAndExp(123456, -3).String()},
		{"1.23456", fromMantAndExp(123456, -5).String()},
		{"0", fromMantAndExp(0, 3).String()},
		{"0", fromMantAndExp(0, 0).String()},
		{"1", fromMantAndExp(10000000000000000, -16).String()},
		{maxMantissaStr, fromMantAndExp(maxMantissa, 0).String()},
		{maxMantissaStr + zeroStr(maxExponent) + addE(maxExponent, false), fromMantAndExp(maxMantissa, maxExponent).String()},
		{
			"0." + zeroStr(maxExponent-len(maxMantissaStr)) + maxMantissaStr + addE(maxExponent-len(maxMantissaStr), true),
			fromMantAndExp(maxMantissa, minExponent).String(),
		},
	}
	for i, item := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(item.expected, item.actual)
		})
	}
}

func TestJSON(t *testing.T) {
	type testItem struct {
		v        Value
		expected []string
	}
	a := assert.New(t)
	mm := uint64(maxMantissa)
	if decimalDigits(mm) > 15 {
		// this is to make float64-conversion precise
		mm /= 1000
	}
	maxMantissaStr := strconv.FormatUint(mm, 10)
	meTemplate := `{"m":%d,"e":%d}`

	modes := []int{JSONModeString, JSONModeFloat, JSONModeME, JSONModeCompact}

	tests := []testItem{
		{fromMantAndExp(0, maxExponent), []string{`"0"`, "0", fmt.Sprintf(meTemplate, 0, 0), `"0"`}},
		{fromMantAndExp(0, minExponent), []string{`"0"`, "0", fmt.Sprintf(meTemplate, 0, 0), `"0"`}},
		{
			fromMantAndExp(mm, 0),
			[]string{
				`"` + maxMantissaStr + `"`,
				maxMantissaStr,
				fmt.Sprintf(meTemplate, mm, 0),
				`"` + maxMantissaStr + `"`,
			},
		},
		{
			fromMantAndExp(123456, 1),
			[]string{
				`"1234560"`,
				"1234560",
				fmt.Sprintf(meTemplate, 123456, 1),
				`"1234560"`,
			},
		},
		{
			fromMantAndExp(123456, -1),
			[]string{
				`"12345.6"`,
				"12345.6",
				fmt.Sprintf(meTemplate, 123456, -1),
				`"12345.6"`,
			},
		},
		{
			fromMantAndExp(123456, 18),
			[]string{
				`"123456000000000000000000"`,
				"123456000000000000000000",
				fmt.Sprintf(meTemplate, 123456, 18),
				fmt.Sprintf(meTemplate, 123456, 18),
			},
		},
		{
			fromMantAndExp(123456, -18),
			[]string{
				`"0.000000000000123456"`,
				"0.000000000000123456",
				fmt.Sprintf(meTemplate, 123456, -18),
				fmt.Sprintf(meTemplate, 123456, -18),
			},
		},
	}
	for i, item := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			shortest, shortestIdx, compactModeLen := 0, -1, 0
			for i, mode := range modes {
				if i >= len(item.expected) {
					break
				}
				data := item.v.toJSON(mode)
				if mode != JSONModeFloat { // this mode is not used when JSONModeCompact is set
					if shortestIdx == -1 || len(data) <= shortest {
						shortest = len(data)
						shortestIdx = i
					}
				}
				if mode == JSONModeCompact {
					compactModeLen = len(data)
				}
				a.Equal(item.expected[i], string(data), "marshalled value error for mode %v", mode)
				var v Value
				if a.NoError(json.Unmarshal(data, &v)) {
					if mode == JSONModeFloat {
						a.InDelta(item.v.Float64(), v.Float64(), 1e10)
					} else {
						a.Equalf(item.v.Normalized(), v, "unmarshalled value error for mode %v", mode)
					}
				}
			}
			if len(item.expected) > JSONModeCompact {
				if shortestIdx != JSONModeCompact {
					t.Errorf("shortest mode was not shortest. was %d with len = %d instead of %d", shortestIdx, shortest, compactModeLen)
				}
			}
		})
	}
}

func TestUnmarshalJSON(t *testing.T) {
	type testItem struct {
		json     string
		err      bool
		expected Value
	}
	a := assert.New(t)

	tests := []testItem{
		{"", true, zero},
		{"{invalid", true, zero},
		{"1234..44", true, zero},
		{`"1234..44"`, true, zero},

		{"1234.5", false, fromMantAndExp(12345, -1)},
		{`"1234.5"`, false, fromMantAndExp(12345, -1)},
		{`"12345e-1"`, false, fromMantAndExp(12345, -1)},
	}

	for i, item := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			var v Value
			err := json.Unmarshal([]byte(item.json), &v)
			if item.err {
				a.Error(err)
			} else {
				a.NoError(err)
				a.Equal(item.expected, v)
			}
		})
	}
}

func TestMantUint64(t *testing.T) {
	type testItem struct {
		v    Value
		exp  int
		mant number
	}
	a := assert.New(t)
	tests := []testItem{
		{fromMantAndExp(123456, -10), -12, 12345600},
		{fromMantAndExp(123456, -5), -5, 123456},
		{fromMantAndExp(12345, -4), -5, 123450},
		{fromMantAndExp(1234, -3), -5, 123400},
		{fromMantAndExp(0, 0), maxExponent + 1, maxMantissa},
		{fromMantAndExp(0, 0), minExponent - 1, 0},
		{fromMantAndExp(maxMantissa, 5), 10, maxMantissa / 100000},
		{fromMantAndExp(maxMantissa, 5), 4, maxMantissa},
		{fromMantAndExp(maxMantissa, 0), uint64Len(maxMantissa) - 1, maxMantissa / uint64(math.Pow10(uint64Len(maxMantissa)-1))},
		{fromMantAndExp(maxMantissa, 0), uint64Len(maxMantissa), 0},
	}
	for i, item := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(item.mant, item.v.ToExp(item.exp).MantUint64())
		})
	}
}

func TestEq(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a, b Value
		eq   bool
	}{
		{0, 0, true},
		{fromMantAndExp(123456, 5), fromMantAndExp(123456, 5), true},
		{fromMantAndExp(123456, -5), fromMantAndExp(123456, -5), true},
		{fromMantAndExp(12345600, 5), fromMantAndExp(123456, 7), true},
		{fromMantAndExp(123456, -5), fromMantAndExp(12345600, -7), true},
		{Min, fromMantAndExp(minMantissa, minExponent), true},
		{Max, fromMantAndExp(maxMantissa, maxExponent), true},

		{fromMantAndExp((maxMantissa/100)*100, 5), fromMantAndExp(maxMantissa/100, 7), true},
		{fromMantAndExp((maxMantissa/100)*100, minExponent), fromMantAndExp(maxMantissa/100, minExponent+2), true},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equalf(test.eq, test.a.Eq(test.b), "%#v %#v", test.a, test.b)
			a.Equalf(test.eq, test.b.Eq(test.a), "%#v %#v", test.a, test.b)
		})
	}
}

func TestCmp(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a, b Value
		cmp  int
	}{
		{fromMantAndExp(0, 5), fromMantAndExp(0, -5), 0},
		{fromMantAndExp(123456, 5), fromMantAndExp(123456, 5), 0},
		{fromMantAndExp(123457, 5), fromMantAndExp(123456, 5), 1},
		{fromMantAndExp(123456, -5), fromMantAndExp(123456, -5), 0},
		{fromMantAndExp(12345600, 5), fromMantAndExp(123456, 7), 0},
		{fromMantAndExp(12345700, 5), fromMantAndExp(123456, 7), 1},
		{fromMantAndExp(123456, -5), fromMantAndExp(12345600, -7), 0},
		{fromMantAndExp(123456, -5), fromMantAndExp(12345700, -7), -1},

		{fromMantAndExp((maxMantissa/100)*100, 5), fromMantAndExp(maxMantissa/100, 7), 0},
		{fromMantAndExp((maxMantissa/100)*100, minExponent), fromMantAndExp(maxMantissa/100, minExponent+2), 0},

		{fromMantAndExp(123456, 5), fromMantAndExp(123450, 5), 1},

		{fromMantAndExp(123456, 5), fromMantAndExp(12345, 6), 1},
		{Min, fromMantAndExp(2, minExponent), -1},

		{fromMantAndExp(maxMantissa, 0), fromMantAndExp(maxMantissa-1, 0), 1},
		{fromMantAndExp(maxMantissa, maxExponent), fromMantAndExp(maxMantissa-1, maxExponent), 1},
		{fromMantAndExp(maxMantissa, minExponent), fromMantAndExp(maxMantissa-1, minExponent), 1},

		{0, 0, 0},
		{fromMantAndExp(0, 100), fromMantAndExp(0, -100), 0},
		{fromMantAndExp(0, 100), fromMantAndExp(1, -100), -1},
		{fromMantAndExp(0, 100), fromMantAndExp(0, -100), 0},

		{fromMantAndExp(maxMantissa/10, 0), fromMantAndExp(maxMantissa, -1), -1},
		{fromMantAndExp(maxMantissa/100, 0), fromMantAndExp(maxMantissa, -2), -1},

		{fromMantAndExp(123, 5), fromMantAndExp(1234, 6), -1},
		{fromMantAndExp(123, 5), fromMantAndExp(1234, 5), -1},
		{fromMantAndExp(1234, 6), fromMantAndExp(1234, 5), 1},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(test.cmp, test.a.Cmp(test.b))
			a.Equal(-test.cmp, test.b.Cmp(test.a))
		})
	}
}

func TestAdd(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a, b, result Value
	}{
		{zero, zero, zero},
		{fromMantAndExp(0, 1), zero, zero},
		{fromMantAndExp(1234567, 0), zero, fromMantAndExp(1234567, 0)},
		{fromMantAndExp(1234567, 0), fromMantAndExp(1234567, 0), fromMantAndExp(1234567*2, 0)},

		{fromMantAndExp(12345, 5), fromMantAndExp(123450, 4), fromMantAndExp(12345*2, 5)},
		{fromMantAndExp(12345, 5), fromMantAndExp(123456, 4), fromMantAndExp(123450+123456, 4)},

		{fromMantAndExp(maxMantissa/100, 0), fromMantAndExp(maxMantissa/100, 1), fromMantAndExp(maxMantissa/100+(maxMantissa/100)*10, 0)},
		{fromMantAndExp(maxMantissa-8, 10), fromMantAndExp(8, 8), fromMantAndExp(maxMantissa-8, 10)},
		{fromMantAndExp(maxMantissa/100, 10), fromMantAndExp(8, 8), fromMantAndExp((maxMantissa/100)*100+8, 8)},
		{fromMantAndExp(maxMantissa, 0), fromMantAndExp(maxMantissa, 8), fromMantAndExp((maxMantissa+maxMantissa/number(1e8))/10, 9)},

		{fromMantAndExp(maxMantissa, 0), fromMantAndExp(maxMantissa, 0), fromMantAndExp((maxMantissa*2)/10, 1)},
		{fromMantAndExp(maxMantissa-1, maxExponent), fromMantAndExp(maxMantissa, maxExponent), Max},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(test.result, test.a.Add(test.b))
			a.Equal(test.result, test.b.Add(test.a))
		})
	}
}

func TestMul(t *testing.T) {
	a := assert.New(t)
	onlyMant := func(m uint64, _ int) uint64 { return m }
	tests := []struct {
		a, b, result Value
	}{
		{zero, zero, zero},
		{fromMantAndExp(0, 1), zero, zero},
		{fromMantAndExp(1234567, 0), zero, zero},
		{fromMantAndExp(1234567, 0), fromMantAndExp(1234567, 0), fromMantAndExp(1234567*1234567, 0)},
		{fromMantAndExp(1234560, 0), fromMantAndExp(123456, -5), fromMantAndExp(15241383936, -4)},
		{fromMantAndExp(1, maxExponent), fromMantAndExp(1, 4), fromMantAndExp(10000, maxExponent)},
		{
			fromMantAndExp(maxMantissa, 0),
			fromMantAndExp(1e9, 0),
			fromMantAndExp(maxMantissa, 9),
		},
		{
			fromMantAndExp(maxMantissa, 1),
			fromMantAndExp(1e9, 0),
			fromMantAndExp(maxMantissa, 10),
		},
		{fromMantAndExp(1, minExponent), fromMantAndExp(1, -1), zero},
		{fromMantAndExp(1, minExponent), fromMantAndExp(123456, -3), fromMantAndExp(123, minExponent)},
		{
			fromMantAndExp(maxMantissa, minExponent),
			fromMantAndExp(1, -10),
			fromMantAndExp(maxMantissa/uint64(1e10), minExponent),
		},
		{fromMantAndExp(maxMantissa, maxExponent), fromMantAndExp(maxMantissa, 0), Max},
		{fromMantAndExp(maxMantissa/10, 1), fromMantAndExp(6, 0), fromMantAndExp((maxMantissa/10)*6, 1)},
		{fromMantAndExp(maxMantissa/10, 1), fromMantAndExp(20, 0), adjustMantExp(onlyMant(mul64((maxMantissa/10)*10, 2)), 1)},
		{fromMantAndExp(111111111111111, 0), fromMantAndExp(500, maxExponent), fromMantAndExp(55555555555555500, maxExponent)},
		{fromMantAndExp(10000000000000000, 5), fromMantAndExp(maxMantissa, 5), fromMantAndExp(maxMantissa, 26)},
		{fromMantAndExp(1, minExponent), fromMantAndExp(1, -1), zero},
		{fromMantAndExp(10, minExponent), fromMantAndExp(1, -1), fromMantAndExp(1, minExponent)},
		{fromMantAndExp(maxMantissa, minExponent), fromMantAndExp(1, -10), fromMantAndExp(maxMantissa/number(1e10), minExponent)},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(test.result, test.a.Mul(test.b))
			a.Equal(test.result, test.b.Mul(test.a))
		})
	}
}

func TestDiv(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a, b   Value
		panics bool

		div              Value
		divModQ, divModR Value
		prec             int
	}{
		{a: zero, b: zero, panics: true},
		{a: fromMantAndExp(0, 1), b: zero, panics: true},
		{a: zero, b: fromMantAndExp(1234567, 0)},
		{
			a: fromMantAndExp(6, -1), b: fromMantAndExp(15, -2),
			div:     fromMantAndExp(4, 0),
			divModQ: fromMantAndExp(4, 0), divModR: zero, prec: 1,
		},
		{
			a: fromMantAndExp(600, 0), b: fromMantAndExp(12, 0),
			div:     fromMantAndExp(5, 1),
			divModQ: fromMantAndExp(5, 1), divModR: zero, prec: 1,
		},
		{
			a: fromMantAndExp(123, 0), b: fromMantAndExp(125, -3),
			div:     fromMantAndExp(984, 0),
			divModQ: fromMantAndExp(984, 0), divModR: zero, prec: 1,
		},
		{
			a: fromMantAndExp(123, 0), b: fromMantAndExp(7, 0),
			div:     fromMantAndExp(17571428571428572, -15),
			divModQ: fromMantAndExp(17, 0), divModR: fromMantAndExp(4, 0), prec: 0,
		},
		{
			a: fromMantAndExp(15, -3), b: fromMantAndExp(7, 0),
			div:     fromMantAndExp(2142857142857143, -18),
			divModQ: fromMantAndExp(2, -3), divModR: fromMantAndExp(1, -3), prec: 3,
		},
		{
			a: fromMantAndExp(15, -3), b: fromMantAndExp(7, 0),
			div:     fromMantAndExp(2142857142857143, -18),
			divModQ: zero, divModR: fromMantAndExp(15, -3), prec: 2,
		},
		{
			a: fromMantAndExp(123, 0), b: fromMantAndExp(7, 0),
			div:     fromMantAndExp(17571428571428572, -15),
			divModQ: fromMantAndExp(175, -1), divModR: fromMantAndExp(5, -1), prec: 1,
		},
		{
			a: fromMantAndExp(maxMantissa/1000, 0), b: fromMantAndExp(1000, 0),
			div:     fromMantAndExp((maxMantissa / 1000), -3),
			divModQ: fromMantAndExp(maxMantissa/1000, -3), divModR: zero, prec: 1,
		},
		{
			a: fromMantAndExp(10, 0), b: fromMantAndExp(3, 0),
			div:     fromMantAndExp(3333333333333333, -15),
			divModQ: fromMantAndExp(3, 0), divModR: fromMantAndExp(1, 0), prec: 0,
		},
		{
			a: fromMantAndExp(10, 0), b: fromMantAndExp(3, 0),
			div:     fromMantAndExp(3333333333333333, -15),
			divModQ: fromMantAndExp(333333, -5), divModR: fromMantAndExp(1, -5), prec: 5,
		},
		{
			a: fromMantAndExp(15, 0), b: fromMantAndExp(3, minExponent),
			div:     fromMantAndExp(5, maxExponent),
			divModQ: fromMantAndExp(5, maxExponent), divModR: zero, prec: 5,
		},
		{
			a: fromMantAndExp(15, 0), b: fromMantAndExp(7, minExponent),
			div:     fromMantAndExp(21428571428571424, expType(maxExponent-uint64Len(21428571428571424)+1)),
			divModQ: fromMantAndExp(21428571, expType(maxExponent-uint64Len(21428571)+1)), divModR: fromMantAndExp(3, -7), prec: -11,
		},
		{
			a: fromMantAndExp(15, 0), b: fromMantAndExp(17, 0),
			div:     fromMantAndExp(8823529411764706, -16),
			divModQ: zero, divModR: fromMantAndExp(15, 0), prec: -1,
		},
		{
			a: fromMantAndExp(134, 0), b: fromMantAndExp(3, 0),
			div:     adjustMantExp(44666666666666664, -15),
			divModQ: fromMantAndExp(4, 1), divModR: fromMantAndExp(14, 0), prec: -1,
		},
		{
			a: fromMantAndExp(134, 0), b: fromMantAndExp(3, 0),
			div:     adjustMantExp(44666666666666664, -15),
			divModQ: fromMantAndExp(44, 0), divModR: fromMantAndExp(2, 0), prec: 0,
		},
		{
			a: fromMantAndExp(134, 0), b: fromMantAndExp(3, 0),
			div:     adjustMantExp(44666666666666664, -15),
			divModQ: zero, divModR: fromMantAndExp(134, 0), prec: -5,
		},
		{
			a: fromMantAndExp(15, -2), b: fromMantAndExp(7, 0),
			div:     fromMantAndExp(2142857142857143, -17),
			divModQ: zero, divModR: fromMantAndExp(15, -2), prec: 0,
		},
		{
			a: fromMantAndExp(15, -2), b: fromMantAndExp(7, 0),
			div:     fromMantAndExp(2142857142857143, -17),
			divModQ: zero, divModR: fromMantAndExp(15, -2), prec: 1,
		},
		{
			a: fromMantAndExp(15, -2), b: fromMantAndExp(7, 0),
			div:     fromMantAndExp(2142857142857143, -17),
			divModQ: zero, divModR: fromMantAndExp(15, -2), prec: 1,
		},
		{
			a: fromMantAndExp(15, -2), b: fromMantAndExp(7, 0),
			div:     fromMantAndExp(2142857142857143, -17),
			divModQ: fromMantAndExp(2, -2), divModR: fromMantAndExp(1, -2), prec: 2,
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if test.panics {
				a.Panics(func() {
					test.a.Div(test.b)
				})
				return
			}
			div := test.a.Div(test.b)
			if div.IsZero() || test.b.IsZero() {
				a.Equal(test.div, div, "%s / %s (diff = %s)", test.a.String(), test.b.String())
			} else {
				diff, _ := test.div.sub(div)
				a.True(diff.Div(test.div).Cmp(MustFromString("0.0000001")) < 0)
			}
			q, r := test.a.DivMod(test.b, test.prec)
			a.Equal(test.divModQ, q, "%s / %s", test.a.String(), test.b.String())
			a.Equal(test.divModR, r, "%s / %s", test.a.String(), test.b.String())
			a.Equal(test.a.Normalized(), test.b.Mul(test.divModQ).Add(test.divModR).Normalized(),
				"%s / %s", test.a.String(), test.b.String())
			a.InDelta(test.a.Normalized().Float64(), test.b.Mul(div).Normalized().Float64(),
				1e-13, "%s / %s", test.a.String(), test.b.String())
		})
	}
}

func TestDecimalDigits(t *testing.T) {
	a := assert.New(t)
	tests := []uint64{0, 1, 9, 10, 11, 100, 1000, 1e10, maxMantissa, math.MaxUint64}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(uint64Len(test), decimalDigits(test))
		})
	}
}

func BenchmarkEq(b *testing.B) {
	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < b.N; i++ {
		v1 := fromMantAndExp(number(rnd.Uint32()), expType(rnd.Int31n(maxExponent)-maxExponent/2))
		v2 := fromMantAndExp(number(rnd.Uint32()), expType(rnd.Int31n(maxExponent)-maxExponent/2))
		v1.Eq(v2)
	}
}

func BenchmarkCmp(b *testing.B) {
	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < b.N; i++ {
		v1 := fromMantAndExp(number(rnd.Uint32()), expType(rnd.Int31n(maxExponent)-maxExponent/2))
		v2 := fromMantAndExp(number(rnd.Uint32()), expType(rnd.Int31n(maxExponent)-maxExponent/2))
		v1.Cmp(v2)
	}
}

func uint64Len(value uint64) int {
	result := 1
	for value > 9 {
		value /= 10
		result++
	}
	return result
}

func cutToDigits(v uint64, digits int) (uint64, int) {
	dd := decimalDigits(v)
	diff := dd - digits
	if diff <= 0 {
		return v, 0
	}
	v /= pow10(diff)
	return v, diff
}
