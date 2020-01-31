// Copyright 2020 Aleksandr Demakin. All rights reserved.

package fixed

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"testing"

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
		{0.012345, fromMantAndExp(12345, -6), ""},
		{123450000, fromMantAndExp(12345, 4), ""},
		{maxMantissa / 100, fromMantAndExp(maxMantissa/100, 0), ""},
		{0.12345e-50, fromMantAndExp(12345, -55), ""},
		{1e127, fromMantAndExp(1, maxExponent), ""},
		{1e-127, fromMantAndExp(1, minExponent), ""},
		{1e128, fromMantAndExp(10, 127), ""},

		{1e-128, fromMantAndExp(maxMantissa, 0), "value out of range"},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			v, err := FromFloat64(test.f)
			if len(test.err) == 0 {
				if a.NoError(err) {
					a.Equal(test.v, v)
				}
			} else {
				a.EqualError(err, test.err)
			}
		})
	}
}

func TestFromString(t *testing.T) {
	a := assert.New(t)
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
		{"0.012345", fromMantAndExp(12345, -6), ""},
		{"123450000", fromMantAndExp(12345, 4), ""},
		{strconv.FormatUint(maxMantissa, 10), fromMantAndExp(maxMantissa, 0), ""},
		{strconv.FormatUint(maxMantissa/100, 10), fromMantAndExp(maxMantissa/100, 0), ""},
		{"0.12345e-50", fromMantAndExp(12345, -55), ""},
		{"1e128", fromMantAndExp(10, 127), ""},

		{"", zero, "empty input"},
		{`"`, zero, "empty input"},
		{`  ""  `, zero, "parsing failed: unexpected symbol '\"' at pos 3"},
		{`"   -"`, zero, "negative value"},
		{`abc`, zero, "parsing failed: unexpected symbol 'a' at pos 1"},
		{`  "abc`, zero, "parsing failed: unexpected symbol '\"' at pos 3"},
		{`   0.00.  `, zero, "parsing failed: unexpected delimeter at pos 8"},
		{"1e-128", zero, "value out of range"},
		{"1" + manyZeros, zero, "value out of range"},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			v, err := FromString(test.s)
			if len(test.e) == 0 {
				if a.NoError(err) {
					a.Equal(test.v, v)
				}
			} else {
				a.EqualError(err, test.e)
			}
		})
	}
}

func TestFromMantAndExp(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		m, mExpected number
		e            int8
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

func TestNormalize(t *testing.T) {
	type testItem struct {
		expected, actual Value
	}
	a := assert.New(t)
	tests := []testItem{
		{zero, fromMantAndExp(0, 0).Normalized()},
		{zero, fromMantAndExp(0, 3).Normalized()},
		{fromMantAndExp(123, 2), fromMantAndExp(12300, 0).Normalized()},
		{fromMantAndExp(123, 3), fromMantAndExp(12300, 1).Normalized()},
		{fromMantAndExp(123, 0), fromMantAndExp(123, 0).Normalized()},
		{fromMantAndExp(123, 0), fromMantAndExp(123000000000, -9).Normalized()},
		{fromMantAndExp(123, 5), fromMantAndExp(123000000000, -4).Normalized()},
		{fromMantAndExp(123456789, 0), fromMantAndExp(123456789, 0).Normalized()},
		{fromMantAndExp(123456789, 4), fromMantAndExp(123456789, 4).Normalized()},
		{fromMantAndExp(12345, minExponent+4), fromMantAndExp(123450000, minExponent).Normalized()},

		{fromMantAndExp(12345, maxExponent), fromMantAndExp(123450, maxExponent-1).Normalized()},
		{fromMantAndExp(123450, maxExponent), fromMantAndExp(1234500, maxExponent-1).Normalized()},
		{fromMantAndExp(1, minExponent+16), fromMantAndExp(1e16, minExponent).Normalized()},
	}
	for i, item := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(item.expected, item.actual)
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
		{123 * 1e127, fromMantAndExp(123, maxExponent).Float64()},
		{0, fromMantAndExp(0, 3).Float64()},
		{0, fromMantAndExp(0, 0).Float64()},
		{7.205759403792793e+143, fromMantAndExp(maxMantissa, maxExponent).Float64()},
		{7.205759403792794e-111, fromMantAndExp(maxMantissa, minExponent).Float64()},
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
		{"72057594037927935", fromMantAndExp(maxMantissa, 0).String()},
		{fmt.Sprintf("%d", maxMantissa) + manyZeros[:127], fromMantAndExp(maxMantissa, maxExponent).String()},
		{"0." + manyZeros[:110] + fmt.Sprintf("%d", maxMantissa), fromMantAndExp(maxMantissa, minExponent).String()},
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
	maxMantissaStr := strconv.FormatUint(maxMantissa, 10)
	meTemplate := `{"m":%d,"e":%d}`

	modes := []int{JSONModeString, JSONModeFloat, JSONModeME, JSONModeCompact}

	tests := []testItem{
		{fromMantAndExp(0, maxExponent), []string{`"0"`, "0", fmt.Sprintf(meTemplate, 0, 0), `"0"`}},
		{fromMantAndExp(0, minExponent), []string{`"0"`, "0", fmt.Sprintf(meTemplate, 0, 0), `"0"`}},
		{
			fromMantAndExp(maxMantissa, 0),
			[]string{
				`"` + maxMantissaStr + `"`,
				"72057594037927940",
				fmt.Sprintf(meTemplate, maxMantissa, 0),
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
		{fromMantAndExp(0, 0), maxExponent + 1, Max},
		{fromMantAndExp(0, 0), minExponent - 1, 0},
		{fromMantAndExp(maxMantissa, 5), 10, maxMantissa / 100000},
		{fromMantAndExp(maxMantissa, 5), 4, maxMantissa},
		{fromMantAndExp(maxMantissa, 0), uint64Len(maxMantissa) - 1, maxMantissa / uint64(1e16)},
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

		{fromMantAndExp((maxMantissa/100)*100, 5), fromMantAndExp(maxMantissa/100, 7), true},
		{fromMantAndExp((maxMantissa/100)*100, minExponent), fromMantAndExp(maxMantissa/100, minExponent+2), true},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(test.eq, test.a.Eq(test.b))
		})
	}
}
