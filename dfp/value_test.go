// Copyright 2020 Aleksandr Demakin. All rights reserved.

package dfp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/avdva/numeric/internal/mathutil"

	"github.com/stretchr/testify/assert"
)

func TestFromFloat(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		f   float64
		v   Value
		err string
	}{
		{0, zero, ""},
		{0.012345, combine(12345, -6), ""},
		{123450000, combine(12345, 4), ""},
		{maxMantissa / 100, combine(maxMantissa/100, 0), ""},
		{float64(0.12345) * math.Pow10(minExponent+5), combine(12345, minExponent), ""},
		{math.Pow10(maxExponent), combine(1, maxExponent), ""},
		{math.Pow10(minExponent), combine(1, minExponent), ""},
		{math.Pow10(maxExponent + 1), combine(10, maxExponent), ""},
		{math.Pow10(minExponent - 1), zero, ""},
		{float64(15) / 7, adjustMantExp(21428571428571428, -16), ""},

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
		{"00001.10000", combine(11, -1), ""},
		{"+000010.01000", combine(1001, -2), ""},
		{zeroStr(100) + strconv.FormatUint(maxMantissa, 10) + "." + zeroStr(100), combine(maxMantissa, 0), ""},
		{strconv.FormatUint(maxMantissa, 10) + "." + zeroStr(100), combine(maxMantissa, 0), ""},
		{"0.01234500", combine(12345, -6), ""},
		{"123450000", combine(12345, 4), ""},
		{"123450000.", combine(12345, 4), ""},
		{"0123450000.", combine(12345, 4), ""},
		{"0123450000.0", combine(12345, 4), ""},
		{".123450000", combine(12345, -5), ""},
		{"0.123450000", combine(12345, -5), ""},
		{"012.3450", combine(12345, -3), ""},
		{"12.345", combine(12345, -3), ""},
		{strconv.FormatUint(maxMantissa, 10), combine(maxMantissa, 0), ""},
		{strconv.FormatUint(maxMantissa/100, 10), combine(maxMantissa/100, 0), ""},
		{"0.12345e" + strconv.Itoa(minExponent+5), combine(12345, minExponent), ""},
		{"1e" + strconv.Itoa(maxExponent+1), combine(10, maxExponent), ""},
		{"1e" + strconv.FormatInt(minExponent-1, 10), zero, ""},
		{"123e" + strconv.Itoa(maxExponent+20), Max, ""},
		{testNumStr + "0", combine(dd, expType(1+e)), ""},
		{"000" + testNumStr + "00000", combine(dd, expType(5+e)), ""},
		{"000" + testNumStr + "00000.00000000", combine(dd, expType(5+e)), ""},
		{testNumStr + zeroStr(-minExponent-10), combine(dd, -minExponent-10+expType(e)), ""},
		{"." + zeroStr(-minExponent-10) + testNumStr, combine(dd/mathutil.Pow10(9), minExponent), ""},
		{"123e10", FromMantAndExp(123, 10), ""},
		{"123e-10", FromMantAndExp(123, -10), ""},
		{"", zero, "empty input"},
		{`"`, zero, "empty input"},
		{`  ""  `, zero, "parsing failed: unexpected symbol '\"' at pos 3"},
		{`"   -"`, zero, "empty input"},
		{`"   --"`, zero, "negative value"},
		{`"   +---"`, zero, "parsing failed: unexpected symbol '-' at pos 6"},
		{`abc`, zero, "parsing failed: unexpected symbol 'a' at pos 1"},
		{`  "abc`, zero, "parsing failed: unexpected symbol '\"' at pos 3"},
		{`   0.00.  `, zero, "parsing failed: unexpected delimeter at pos 8"},
		{"123e", zero, "parsing failed: error parsing exponent: strconv.ParseInt: parsing \"\": invalid syntax at pos 5"},
		{"123e-t5", zero, "parsing failed: error parsing exponent: strconv.ParseInt: parsing \"-t5\": invalid syntax at pos 5"},
		{"123e1" + zeroStr(20), zero, "parsing failed: error parsing exponent: strconv.ParseInt: parsing \"100000000000000000000\": value out of range at pos 5"},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			v, err := FromString(test.s)
			if len(test.e) == 0 {
				if a.NoError(err) {
					a.Equal(test.v, v, test.s)
				}
			} else {
				a.Panics(func() {
					MustFromString(test.s)
				})
				a.EqualError(err, test.e)
			}
		})
	}
}

func TestFromMantAndExp(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		m, mExpected number
		e            expType
	}{
		{0, 0, 0},
		{123456, 123456, 0},
		{123456, 123456, maxExponent},
		{maxMantissa, maxMantissa, maxExponent},
		{maxMantissa, maxMantissa, minExponent},
		{maxMantissa, maxMantissa, 1},
		{maxMantissa, maxMantissa, -1},
		{maxMantissa + 1, 0, minExponent},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			v := combine(test.m, test.e)
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
		{123456, true, combine(123456, 0)},
		{maxMantissa, true, combine(maxMantissa, 0)},
		{1234, false, combine(123456, -2)},
		{0, true, combine(0, 0)},
		{0, true, combine(0, 127)},
		{1, true, combine(1000, -3)},
		{12, true, combine(12000, -3)},
		{maxMantissa / uint64(1e10), false, combine(maxMantissa, -10)},
		{1, true, combine(1000, -3)},
		{maxMantissa, false, combine(maxMantissa, 1)},
		{maxMantissa, false, combine(maxMantissa-1, 1)},
		{maxMantissa, false, combine(maxMantissa/10+1, 1)},
		{(maxMantissa / 100) * 10, true, combine(maxMantissa/100, 1)},
		{(maxMantissa / 10) * 10, true, combine(maxMantissa/10, 1)},
	}
	for i, item := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			u, exact := item.v.Uint64()
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
		{zero, combine(0, 0)},
		{zero, combine(0, 3)},
		{combine(123, 2), combine(12300, 0)},
		{combine(123, 3), combine(12300, 1)},
		{combine(123, 0), combine(123, 0)},
		{combine(123, 0), combine(123000000000, -9)},
		{combine(123, 5), combine(123000000000, -4)},
		{combine(123456789, 0), combine(123456789, 0)},
		{combine(123456789, 4), combine(123456789, 4)},
		{combine(12345, minExponent+4), combine(123450000, minExponent)},

		{combine(12345, maxExponent), combine(123450, maxExponent-1)},
		{combine(123450, maxExponent), combine(1234500, maxExponent-1)},
		{combine(1234500, maxExponent), combine(12345000, maxExponent-1)},
		{combine(1, minExponent+16), combine(1e16, minExponent)},
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
			maxMantissa / mathutil.Pow10(digitsInMaxMantissa-1),
			maxExponent,
		},
		{maxMantissa, maxExponent, maxMantissa, maxExponent - expType(digitsInMaxMantissa), maxMantissa, 0, maxExponent},
		{
			maxMantissa,
			minExponent + expType(digitsInMaxMantissa) - 1,
			maxMantissa,
			minExponent,
			maxMantissa,
			maxMantissa / (mathutil.Pow10(digitsInMaxMantissa - 1)),
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
			maxMantissa / mathutil.Pow10(digitsInMaxMantissa-2),
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
		{0, combine(0, 0).Float64()},
		{123456, combine(123456, 0).Float64()},
		{0.123456, combine(123456, -6).Float64()},
		{456.789, combine(456789, -3).Float64()},
		{123 * math.Pow10(maxExponent), combine(123, maxExponent).Float64()},
		{0, combine(0, 3).Float64()},
		{0, combine(0, 0).Float64()},
		{float64(maxMantissa) * math.Pow10(maxExponent), combine(maxMantissa, maxExponent).Float64()},
		{float64(maxMantissa) * math.Pow10(minExponent), combine(maxMantissa, minExponent).Float64()},
	}
	for i, item := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(item.expected, item.actual)
		})
	}
}

func TestString(t *testing.T) {
	type testItem struct {
		expected string
		v        Value
	}
	a := assert.New(t)
	mm := uint64(maxMantissa)
	maxMantissaStr := strconv.FormatUint(mm, 10)
	if len(maxMantissaStr) > maxExponent {
		mm /= mathutil.Pow10(len(maxMantissaStr) - maxExponent)
		maxMantissaStr = maxMantissaStr[:maxExponent]
	}
	addE := func(zeros int, neg bool) string {
		if zeros < 0 {
			return ""
		}
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
		{"123456", combine(123456, 0)},
		{"0.123456", combine(123456, -6)},
		{"0.0000123456", combine(123456, -10)},
		{"0.00123456", combine(123456, -8)},
		{"123.456", combine(123456, -3)},
		{"1.23456", combine(123456, -5)},
		{"0", combine(0, 3)},
		{"0", combine(0, 0)},
		{"1", combine(1000, -3)},
		{maxMantissaStr, combine(mm, 0)},
		{maxMantissaStr + zeroStr(maxExponent) + addE(maxExponent, false), combine(mm, maxExponent)},
		{
			"0." + zeroStr(-minExponent-len(maxMantissaStr)) + maxMantissaStr + addE(-minExponent-len(maxMantissaStr), true),
			combine(mm, minExponent),
		},
	}
	for i, item := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			s := item.v.String()
			a.Equal(item.expected, s)
			a.Equal(item.v.Normalized(), MustFromString(s))
		})
	}
}

func TestFormat(t *testing.T) {
	type testItem struct {
		v    Value
		f, e string
	}
	a := assert.New(t)
	tests := []testItem{
		{combine(0, 0), "0", "0"},
		{combine(0, -1), "0", "0"},
		{combine(0, 1), "0", "0"},
		{combine(1, 3), "1000", "1e3"},
		{combine(1000, 0), "1000", "1e3"},
		{combine(123, 10), "1230000000000", "123e10"},
		{combine(1234560, -3), "1234.56", "123456e-2"},
	}
	for i, item := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			f, e := fmt.Sprintf("%f", item.v), fmt.Sprintf("%e", item.v)
			a.Equal(item.f, f)
			a.Equal(item.e, e)
			n := item.v.Normalized()
			a.Equal(n, MustFromString(f))
			a.Equal(n, MustFromString(e))
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
	me := expType(maxExponent)
	if me > 18 {
		me = 18
	}
	if mathutil.DecimalDigits(mm) > 15 {
		// this is to make float64-conversion precise
		mm /= 1000
	}
	maxMantissaStr := strconv.FormatUint(mm, 10)
	meTemplate := `{"m":%d,"e":%d}`

	modes := []int{FormatString, FormatFloat, FormatJSONObject, JSONModeCompact}

	tests := []testItem{
		{combine(0, maxExponent), []string{`"0"`, "0", fmt.Sprintf(meTemplate, 0, maxExponent), `"0"`}},
		{combine(0, minExponent), []string{`"0"`, "0", fmt.Sprintf(meTemplate, 0, minExponent), `"0"`}},
		{combine(0, 0), []string{`"0"`, "0", fmt.Sprintf(meTemplate, 0, 0), `"0"`}},
		{
			combine(mm, 0),
			[]string{
				`"` + maxMantissaStr + `"`,
				maxMantissaStr,
				fmt.Sprintf(meTemplate, mm, 0),
				`"` + maxMantissaStr + `"`,
			},
		},
		{
			combine(123456, 1),
			[]string{
				`"1234560"`,
				"1234560",
				fmt.Sprintf(meTemplate, 123456, 1),
				`"1234560"`,
			},
		},
		{
			combine(123456, -1),
			[]string{
				`"12345.6"`,
				"12345.6",
				fmt.Sprintf(meTemplate, 123456, -1),
				`"12345.6"`,
			},
		},
		{
			combine(123456, me),
			[]string{
				`"123456` + zeroStr(int(me)) + `"`,
				"123456" + zeroStr(int(me)),
				fmt.Sprintf(meTemplate, 123456, me),
				fmt.Sprintf(meTemplate, 123456, me),
			},
		},
		{
			combine(123456, -me),
			[]string{
				`"0.` + zeroStr(int(me)-6) + `123456"`,
				"0." + zeroStr(int(me)-6) + "123456",
				fmt.Sprintf(meTemplate, 123456, -me),
				fmt.Sprintf(meTemplate, 123456, -me),
			},
		},
	}
	for i, item := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			shortest, shortestIdx, compactModeLen := 0, -1, 0
			for i, mode := range modes {
				data := item.v.toJSON(mode)
				if mode != FormatFloat { // this mode is not used when JSONModeCompact is set
					if shortestIdx == -1 || len(data) <= shortest {
						shortest = len(data)
						shortestIdx = i
					}
				}
				if mode == JSONModeCompact {
					compactModeLen = len(data)
				} else {
					a.Equal(item.expected[i], string(data), "marshalled value error for mode %v", mode)
				}
				var v Value
				if a.NoError(json.Unmarshal(data, &v)) {
					if mode == FormatFloat {
						a.InDelta(item.v.Float64(), v.Float64(), 1e10)
					} else {
						a.Equalf(item.v.Normalized(), v, "unmarshalled value error for mode %v", mode)
					}
				}
			}
			if modes[shortestIdx] != JSONModeCompact {
				t.Errorf("shortest mode was not shortest. was %d with len = %d instead of %d", shortestIdx, shortest, compactModeLen)
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

		{"1234.5", false, combine(12345, -1)},
		{`"1234.5"`, false, combine(12345, -1)},
		{`"12345e-1"`, false, combine(12345, -1)},
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

func TestScaleMant(t *testing.T) {
	type testItem struct {
		v        Value
		exp      int32
		expected Value
		exact    bool
	}
	a := assert.New(t)
	tests := []testItem{
		{combine(123456, -10), -12, combine(12345600, -12), true},
		{combine(123456, -5), -5, combine(123456, -5), true},
		{combine(12345, -4), -5, combine(123450, -5), true},
		{combine(1234, -3), -5, combine(123400, -5), true},
		{combine(0, 0), maxExponent + 1, combine(0, maxExponent), false},
		{combine(0, 0), minExponent - 1, zero, false},
		{combine(maxMantissa, 5), 10, combine(maxMantissa/100000, 10), false},
		{combine(maxMantissa, 5), 4, combine(maxMantissa, 4), false},
		{combine(maxMantissa, 5), 6, combine(maxMantissa/10, 6), false},
		{combine(maxMantissa, 5), 11, combine(maxMantissa/1000000, 11), false},
		{combine(1, maxExponent), maxExponent - 20, combine(maxMantissa, maxExponent-20), false},
		{combine(1, 0), 30, combine(0, 30), false},
		{combine(1, 1), maxExponent + 1, combine(0, maxExponent), false},
		{combine(1, 1), minExponent - 1, combine(maxMantissa, minExponent), false},
		{combine(1, 30), 0, combine(maxMantissa, 0), false},
		{combine(maxMantissa/10, 5), 3, combine(maxMantissa, 3), false},
		{combine(maxMantissa/10, 9), 1, combine(maxMantissa, 1), false},
		{combine(uint64(math.MaxUint64)/1e7, 10), 3, combine(maxMantissa, 3), false},
		{combine(uint64(math.MaxUint64)/1e7, 10), 13, combine(uint64(math.MaxUint64)/1e10, 13), false},
	}
	for i, item := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			scaled, exact := item.v.ScaleMant(item.exp)
			a.Equal(item.expected, scaled, scaled.GoString())
			a.Equal(item.exact, exact, scaled.GoString())
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
		{combine(123456, 5), combine(123456, 5), true},
		{combine(123456, -5), combine(123456, -5), true},
		{combine(12345600, 5), combine(123456, 7), true},
		{combine(123456, -5), combine(12345600, -7), true},
		{Min, combine(minMantissa, minExponent), true},
		{Max, combine(maxMantissa, maxExponent), true},

		{combine((maxMantissa/100)*100, 5), combine(maxMantissa/100, 7), true},
		{combine((maxMantissa/100)*100, minExponent), combine(maxMantissa/100, minExponent+2), true},
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
		{combine(0, 5), combine(0, -5), 0},
		{combine(123456, 5), combine(123456, 5), 0},
		{combine(123457, 5), combine(123456, 5), 1},
		{combine(123456, -5), combine(123456, -5), 0},
		{combine(12345600, 5), combine(123456, 7), 0},
		{combine(12345700, 5), combine(123456, 7), 1},
		{combine(123456, -5), combine(12345600, -7), 0},
		{combine(123456, -5), combine(12345700, -7), -1},

		{combine((maxMantissa/100)*100, 5), combine(maxMantissa/100, 7), 0},
		{combine((maxMantissa/100)*100, minExponent), combine(maxMantissa/100, minExponent+2), 0},

		{combine(123456, 5), combine(123450, 5), 1},

		{combine(123456, 5), combine(12345, 6), 1},
		{Min, combine(2, minExponent), -1},

		{combine(maxMantissa, 0), combine(maxMantissa-1, 0), 1},
		{combine(maxMantissa, maxExponent), combine(maxMantissa-1, maxExponent), 1},
		{combine(maxMantissa, minExponent), combine(maxMantissa-1, minExponent), 1},

		{0, 0, 0},
		{combine(0, 100), combine(0, -100), 0},
		{combine(0, 100), combine(1, -100), -1},
		{combine(0, 100), combine(0, -100), 0},

		{combine(maxMantissa/10, 0), combine(maxMantissa, -1), -1},
		{combine(maxMantissa/100, 0), combine(maxMantissa, -2), -1},

		{combine(123, 5), combine(1234, 6), -1},
		{combine(123, 5), combine(1234, 5), -1},
		{combine(1234, 6), combine(1234, 5), 1},
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
		{combine(0, 1), zero, zero},
		{combine(1234567, 0), zero, combine(1234567, 0)},
		{combine(1234567, 0), combine(1234567, 0), combine(1234567*2, 0)},

		{combine(12345, 5), combine(123450, 4), combine(12345*2, 5)},
		{combine(12345, 5), combine(123456, 4), combine(123450+123456, 4)},

		{combine(maxMantissa/100, 0), combine(maxMantissa/100, 1), combine(maxMantissa/100+(maxMantissa/100)*10, 0)},
		{combine(maxMantissa-8, 10), combine(8, 8), combine(maxMantissa-8, 10)},
		{combine(maxMantissa/100, 10), combine(8, 8), combine((maxMantissa/100)*100+8, 8)},
		{combine(maxMantissa, 0), combine(maxMantissa, 8), combine((maxMantissa+maxMantissa/number(1e8))/10, 9)},

		{combine(maxMantissa, 0), combine(maxMantissa, 0), combine((maxMantissa*2)/10, 1)},
		{combine(maxMantissa-1, maxExponent), combine(maxMantissa, maxExponent), Max},
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
	onlyMant := func(m uint64, _ int32) uint64 { return m }
	tests := []struct {
		a, b, result Value
	}{
		{zero, zero, zero},
		{combine(0, 1), zero, zero},
		{combine(1234567, 0), zero, zero},
		{combine(1234567, 0), combine(1234567, 0), combine(1234567*1234567, 0)},
		{combine(1234560, 0), combine(123456, -5), combine(15241383936, -4)},
		{combine(1, maxExponent), combine(1, 4), combine(10000, maxExponent)},
		{
			combine(maxMantissa, 0),
			combine(1e9, 0),
			combine(maxMantissa, 9),
		},
		{
			combine(maxMantissa, 1),
			combine(1e9, 0),
			combine(maxMantissa, 10),
		},
		{combine(1, minExponent), combine(1, -1), zero},
		{combine(1, minExponent), combine(123456, -3), combine(123, minExponent)},
		{
			combine(maxMantissa, minExponent),
			combine(1, -10),
			combine(maxMantissa/uint64(1e10), minExponent),
		},
		{combine(maxMantissa, maxExponent), combine(maxMantissa, 0), Max},
		{combine(maxMantissa/10, 1), combine(6, 0), combine((maxMantissa/10)*6, 1)},
		{combine(maxMantissa/10, 1), combine(20, 0), adjustMantExp(onlyMant(mathutil.Mul64((maxMantissa/10)*10, 2)), 1)},
		{combine(111111111111111, 0), combine(500, maxExponent), combine(55555555555555500, maxExponent)},
		{combine(10000000000000000, 5), combine(maxMantissa, 5), combine(maxMantissa, 26)},
		{combine(1, minExponent), combine(1, -1), zero},
		{combine(10, minExponent), combine(1, -1), combine(1, minExponent)},
		{combine(maxMantissa, minExponent), combine(1, -10), combine(maxMantissa/number(1e10), minExponent)},
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
		{a: combine(0, 1), b: zero, panics: true},
		{a: zero, b: combine(1234567, 0)},
		{
			a: combine(6, -1), b: combine(15, -2),
			div:     combine(4, 0),
			divModQ: combine(4, 0), divModR: zero, prec: 1,
		},
		{
			a: combine(600, 0), b: combine(12, 0),
			div:     combine(5, 1),
			divModQ: combine(5, 1), divModR: zero, prec: 1,
		},
		{
			a: combine(123, 0), b: combine(125, -3),
			div:     combine(984, 0),
			divModQ: combine(984, 0), divModR: zero, prec: 1,
		},
		{
			a: combine(123, 0), b: combine(7, 0),
			div:     combine(17571428571428572, -15),
			divModQ: combine(17, 0), divModR: combine(4, 0), prec: 0,
		},
		{
			a: combine(15, -3), b: combine(7, 0),
			div:     combine(2142857142857143, -18),
			divModQ: combine(2, -3), divModR: combine(1, -3), prec: 3,
		},
		{
			a: combine(15, -3), b: combine(7, 0),
			div:     combine(2142857142857143, -18),
			divModQ: zero, divModR: combine(15, -3), prec: 2,
		},
		{
			a: combine(123, 0), b: combine(7, 0),
			div:     combine(17571428571428572, -15),
			divModQ: combine(175, -1), divModR: combine(5, -1), prec: 1,
		},
		{
			a: combine(maxMantissa/1000, 0), b: combine(1000, 0),
			div:     combine((maxMantissa / 1000), -3),
			divModQ: combine(maxMantissa/1000, -3), divModR: zero, prec: 1,
		},
		{
			a: combine(10, 0), b: combine(3, 0),
			div:     combine(3333333333333333, -15),
			divModQ: combine(3, 0), divModR: combine(1, 0), prec: 0,
		},
		{
			a: combine(10, 0), b: combine(3, 0),
			div:     combine(3333333333333333, -15),
			divModQ: combine(333333, -5), divModR: combine(1, -5), prec: 5,
		},
		{
			a: combine(15, 0), b: combine(3, minExponent),
			div:     combine(5, -minExponent),
			divModQ: combine(5, -minExponent), divModR: zero, prec: 5,
		},
		{
			a: combine(15, 0), b: combine(7, minExponent),
			div:     combine(214285714285714, expType(-minExponent-uint64Len(214285714285714)+1)),
			divModQ: combine(21428571, expType(-minExponent-uint64Len(21428571)+1)), divModR: combine(3, -7), prec: -11,
		},
		{
			a: combine(15, 0), b: combine(17, 0),
			div:     combine(8823529411764706, -16),
			divModQ: zero, divModR: combine(15, 0), prec: -1,
		},
		{
			a: combine(134, 0), b: combine(3, 0),
			div:     adjustMantExp(44666666666666664, -15),
			divModQ: combine(4, 1), divModR: combine(14, 0), prec: -1,
		},
		{
			a: combine(134, 0), b: combine(3, 0),
			div:     adjustMantExp(44666666666666664, -15),
			divModQ: combine(44, 0), divModR: combine(2, 0), prec: 0,
		},
		{
			a: combine(134, 0), b: combine(3, 0),
			div:     adjustMantExp(44666666666666664, -15),
			divModQ: zero, divModR: combine(134, 0), prec: -5,
		},
		{
			a: combine(15, -2), b: combine(7, 0),
			div:     combine(2142857142857143, -17),
			divModQ: zero, divModR: combine(15, -2), prec: 0,
		},
		{
			a: combine(15, -2), b: combine(7, 0),
			div:     combine(2142857142857143, -17),
			divModQ: zero, divModR: combine(15, -2), prec: 1,
		},
		{
			a: combine(15, -2), b: combine(7, 0),
			div:     combine(2142857142857143, -17),
			divModQ: zero, divModR: combine(15, -2), prec: 1,
		},
		{
			a: combine(15, -2), b: combine(7, 0),
			div:     combine(2142857142857143, -17),
			divModQ: combine(2, -2), divModR: combine(1, -2), prec: 2,
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
				a.Equal(test.div, div, "%s / %s", test.a.String(), test.b.String())
			} else {
				diff, _ := test.div.Sub(div)
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

func TestRound(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a                  Value
		floor, round, ceil Value
		prec               int
	}{
		{zero, zero, zero, zero, 0},
		{zero, zero, zero, zero, 5},
		{zero, zero, zero, zero, -5},

		{
			MustFromString("123.456"),
			MustFromString("123.45"),
			MustFromString("123.46"),
			MustFromString("123.46"),
			2,
		},
		{
			MustFromString("123.456"),
			MustFromString("123.4"),
			MustFromString("123.5"),
			MustFromString("123.5"),
			1,
		},
		{
			MustFromString("123.456"),
			MustFromString("123"),
			MustFromString("123"),
			MustFromString("124"),
			0,
		},
		{
			MustFromString("123.456"),
			MustFromString("120"),
			MustFromString("120"),
			MustFromString("130"),
			-1,
		},
		{
			MustFromString("123.456"),
			MustFromString("100"),
			MustFromString("100"),
			MustFromString("200"),
			-2,
		},
		{
			MustFromString("123.456"),
			MustFromString("0"),
			MustFromString("0"),
			MustFromString("0"),
			-3,
		},
		{
			MustFromString("000.456000"),
			MustFromString(".45"),
			MustFromString(".46"),
			MustFromString(".46"),
			2,
		},
		{
			MustFromString("000.0000123"),
			MustFromString("0"),
			MustFromString("0"),
			MustFromString("1"),
			0,
		},
		{
			MustFromString("000.0000123"),
			MustFromString("0"),
			MustFromString("0"),
			MustFromString("0"),
			-1,
		},
		{
			MustFromString("000.0000123"),
			MustFromString("0"),
			MustFromString("0"),
			MustFromString("0"),
			-3,
		},
		{
			MustFromString("000.0000123"),
			MustFromString("0"),
			MustFromString("0"),
			MustFromString(".1"),
			1,
		},
		{
			MustFromString("000.0000123"),
			MustFromString("0"),
			MustFromString("0"),
			MustFromString("0"),
			-7,
		},
		{
			MustFromString("000.0000123"),
			MustFromString("000.0000123"),
			MustFromString("000.0000123"),
			MustFromString("000.0000123"),
			7,
		},
		{
			MustFromString("000.0000123"),
			MustFromString("000.0000123"),
			MustFromString("000.0000123"),
			MustFromString("000.0000123"),
			8,
		},
		{
			MustFromString("000.0000123"),
			MustFromString("000.000012"),
			MustFromString("000.000012"),
			MustFromString("000.000013"),
			6,
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(test.floor, test.a.Floor(test.prec), "%s (%d)", test.a, test.prec)
			a.Equal(test.round, test.a.Round(test.prec), "%s (%d)", test.a, test.prec)
			a.Equal(test.ceil, test.a.Ceil(test.prec), "%s (%d)", test.a, test.prec)
		})
	}
}

func TestDecimalDigits(t *testing.T) {
	a := assert.New(t)
	tests := []uint64{0, 1, 9, 10, 11, 100, 1000, 1e10, maxMantissa, math.MaxUint64}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(uint64Len(test), mathutil.DecimalDigits(test))
		})
	}
}

func BenchmarkEq(b *testing.B) {
	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < b.N; i++ {
		v1 := combine(number(rnd.Uint32()), expType(rnd.Int31n(maxExponent)-maxExponent/2))
		v2 := combine(number(rnd.Uint32()), expType(rnd.Int31n(maxExponent)-maxExponent/2))
		v1.Eq(v2)
	}
}

func BenchmarkCmp(b *testing.B) {
	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < b.N; i++ {
		v1 := combine(number(rnd.Uint32()), expType(rnd.Int31n(maxExponent)-maxExponent/2))
		v2 := combine(number(rnd.Uint32()), expType(rnd.Int31n(maxExponent)-maxExponent/2))
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
	dd := mathutil.DecimalDigits(v)
	diff := dd - digits
	if diff <= 0 {
		return v, 0
	}
	v /= mathutil.Pow10(diff)
	return v, diff
}

func zeroStr(count int) string {
	return string(bytes.Repeat([]byte{'0'}, count))
}
