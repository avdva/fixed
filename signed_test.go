// Copyright 2020 Aleksandr Demakin. All rights reserved.

package fixed

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSigned_AddValue(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a      Signed
		b      Value
		result Signed
	}{
		{signedZero, zero, signedZero},
		{Signed{Neg: true}, zero, signedZero},
		{Signed{Neg: true, V: MustFromString("1")}, MustFromString("1"), signedZero},
		{Signed{V: MustFromString("12.34")}, MustFromString("4.56"), Signed{V: MustFromString("16.9")}},
		{Signed{Neg: true, V: MustFromString("12.34")}, MustFromString("4.56"), Signed{Neg: true, V: MustFromString("7.78")}},
		{Signed{Neg: true, V: MustFromString("4.56")}, MustFromString("12.34"), Signed{Neg: false, V: MustFromString("7.78")}},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(test.result, test.a.AddValue(test.b).Normalized())
		})
	}
}

func TestSigned_SubValue(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a      Signed
		b      Value
		result Signed
	}{
		{signedZero, zero, signedZero},
		{Signed{Neg: true}, zero, signedZero},
		{Signed{Neg: false, V: MustFromString("1")}, MustFromString("1"), signedZero},
		{Signed{V: MustFromString("12.34")}, MustFromString("4.56"), Signed{V: MustFromString("7.78")}},
		{Signed{Neg: true, V: MustFromString("12.34")}, MustFromString("4.56"), Signed{Neg: true, V: MustFromString("16.9")}},
		{Signed{Neg: false, V: MustFromString("4.56")}, MustFromString("12.34"), Signed{Neg: true, V: MustFromString("7.78")}},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(test.result, test.a.SubValue(test.b).Normalized())
		})
	}
}

func TestSigned_MulValue(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a      Signed
		b      Value
		result Signed
	}{
		{signedZero, zero, signedZero},
		{Signed{Neg: true}, zero, signedZero},
		{Signed{Neg: true, V: MustFromString("1")}, zero, signedZero},
		{Signed{Neg: false, V: MustFromString("1")}, zero, signedZero},
		{Signed{V: MustFromString("12.34")}, MustFromString("2"), Signed{V: MustFromString("24.68")}},
		{Signed{V: MustFromString("12.34"), Neg: true}, MustFromString("2"), Signed{V: MustFromString("24.68"), Neg: true}},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(test.result, test.a.MulValue(test.b).Normalized())
		})
	}
}

func TestSigned_DivValue(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a      Signed
		b      Value
		result Signed
		panics bool
	}{
		{signedZero, zero, signedZero, true},
		{Signed{Neg: true}, zero, signedZero, true},
		{Signed{V: MustFromString("24.68")}, MustFromString("2"), Signed{V: MustFromString("12.34")}, false},
		{Signed{V: MustFromString("24.68"), Neg: true}, MustFromString("2"), Signed{V: MustFromString("12.34"), Neg: true}, false},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if test.panics {
				a.Panics(func() {
					test.a.DivValue(test.b)
				})
			} else {
				a.Equal(test.result, test.a.DivValue(test.b).Normalized())
			}
		})
	}
}

func TestSigned_Add(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a, b   Signed
		result Signed
	}{
		{signedZero, signedZero, signedZero},
		{Signed{Neg: true}, signedZero, signedZero},
		{
			Signed{V: MustFromString("12.34")},
			Signed{V: MustFromString("4.56")},
			Signed{V: MustFromString("16.9")},
		},
		{
			Signed{V: MustFromString("12.34")},
			Signed{V: MustFromString("4.56"), Neg: true},
			Signed{V: MustFromString("7.78")},
		},
		{
			Signed{V: MustFromString("12.34"), Neg: true},
			Signed{V: MustFromString("4.56"), Neg: true},
			Signed{V: MustFromString("16.9"), Neg: true},
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			a.Equal(test.result, test.a.Add(test.b).Normalized())
			a.Equal(test.result, test.b.Add(test.a).Normalized())
		})
	}
}

func TestSigned_Sub(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a, b   Signed
		result Signed
	}{
		{signedZero, signedZero, signedZero},
		{Signed{Neg: true}, signedZero, signedZero},
		{
			Signed{V: MustFromString("12.34")},
			Signed{V: MustFromString("4.56")},
			Signed{V: MustFromString("7.78")},
		},
		{
			Signed{V: MustFromString("12.34")},
			Signed{V: MustFromString("4.56"), Neg: true},
			Signed{V: MustFromString("16.9")},
		},
		{
			Signed{V: MustFromString("12.34"), Neg: true},
			Signed{V: MustFromString("4.56"), Neg: true},
			Signed{V: MustFromString("7.78"), Neg: true},
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			r := test.a.Sub(test.b).Normalized()
			a.Equal(test.result, r)
			r = test.b.Sub(test.a)
			r.Neg = !r.Neg
			r = r.Normalized()
			a.Equal(test.result, r)
		})
	}
}

func TestSigned_Mul(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a, b   Signed
		result Signed
	}{
		{signedZero, signedZero, signedZero},
		{Signed{Neg: true}, signedZero, signedZero},
		{
			Signed{V: MustFromString("12.34")},
			Signed{V: MustFromString("2")},
			Signed{V: MustFromString("24.68")},
		},
		{
			Signed{V: MustFromString("12.34")},
			Signed{V: MustFromString("2"), Neg: true},
			Signed{V: MustFromString("24.68"), Neg: true},
		},
		{
			Signed{V: MustFromString("12.34"), Neg: true},
			Signed{V: MustFromString("2"), Neg: true},
			Signed{V: MustFromString("24.68")},
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			r := test.a.Mul(test.b).Normalized()
			a.Equal(test.result, r)
			r = test.b.Mul(test.a)
			r = r.Normalized()
			a.Equal(test.result, r)
		})
	}
}

func TestSigned_Div(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a, b   Signed
		result Signed
		panics bool
	}{
		{signedZero, signedZero, signedZero, true},
		{Signed{Neg: true}, signedZero, signedZero, true},
		{
			Signed{V: MustFromString("15")},
			Signed{V: MustFromString("3")},
			Signed{V: MustFromString("5")},
			false,
		},
		{
			Signed{V: MustFromString("15")},
			Signed{V: MustFromString("3"), Neg: true},
			Signed{V: MustFromString("5"), Neg: true},
			false,
		},
		{
			Signed{V: MustFromString("15"), Neg: true},
			Signed{V: MustFromString("3"), Neg: true},
			Signed{V: MustFromString("5")},
			false,
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
			r := test.a.Div(test.b).Normalized()
			a.Equal(test.result, r)
			r = test.b.Div(test.a)
			r = NewSigned(MustFromString("1"), false).Div(r).Normalized()
			a.Equal(test.result, r)
		})
	}
}

func TestSigned_DivMod(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		a, b   Signed
		q, r   Signed
		panics bool
	}{
		{signedZero, signedZero, signedZero, signedZero, true},
		{Signed{Neg: true}, signedZero, signedZero, signedZero, true},
		{
			Signed{V: MustFromString("15")},
			Signed{V: MustFromString("7")},
			Signed{V: MustFromString("2")},
			Signed{V: MustFromString("1")},
			false,
		},
		{
			Signed{V: MustFromString("15")},
			Signed{V: MustFromString("7"), Neg: true},
			Signed{V: MustFromString("2"), Neg: true},
			Signed{V: MustFromString("1")},
			false,
		},
		{
			Signed{V: MustFromString("15"), Neg: true},
			Signed{V: MustFromString("7")},
			Signed{V: MustFromString("2"), Neg: true},
			Signed{V: MustFromString("1"), Neg: true},
			false,
		},
		{
			Signed{V: MustFromString("15"), Neg: true},
			Signed{V: MustFromString("7"), Neg: true},
			Signed{V: MustFromString("2")},
			Signed{V: MustFromString("1"), Neg: true},
			false,
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if test.panics {
				a.Panics(func() {
					test.a.DivMod(test.b, 0)
				})
				return
			}
			q, r := test.a.DivMod(test.b, 0)
			a.Equal(test.q, q)
			a.Equal(test.r, r)
			a.Equal(test.a, test.b.Mul(q).Add(r))
		})
	}
}
