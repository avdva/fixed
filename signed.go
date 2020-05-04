package fixed

import "strings"

var (
	signedZero = Signed{false, zero}
)

// Signed is a Value with a sign flag.
type Signed struct {
	Neg bool
	V   Value
}

// NewSigned returns new Signed value.
func NewSigned(v Value, neg bool) Signed {
	return Signed{V: v, Neg: neg}
}

// PosValue converts Value to a signed value.
func PosValue(v Value) Signed {
	return NewSigned(v, false)
}

// NegValue returns a negative signed value for given Value.
func NegValue(v Value) Signed {
	return NewSigned(v, true)
}

// String returns string representation of a signed value.
func (s Signed) String() string {
	var builder strings.Builder
	if s.Neg {
		builder.WriteRune('-')
	}
	s.V.WriteToStringsBuilder(&builder)
	return builder.String()
}

// Add returns a + b.
func (s Signed) Add(other Signed) Signed {
	if s.Neg == other.Neg {
		// v1+v2
		// or -v1+(-v2) = -(v1+v2)
		return NewSigned(s.V.Add(other.V), s.Neg)
	}
	if !s.Neg { // v1+(-v2) = v1-v2
		return NewSigned(s.V.Sub(other.V))
	}
	return NewSigned(other.V.Sub(s.V)) // -v1+v2 = v2-v1
}

// Sub returns a-b.
func (s Signed) Sub(other Signed) Signed {
	other.Neg = !other.Neg
	return s.Add(other) // v1-v2 = v1+(-v2)
}

// Mul returns a*b.
func (s Signed) Mul(other Signed) Signed {
	return NewSigned(s.V.Mul(other.V), s.Neg != other.Neg)
}

// Div calculates a/b. If b == 0, Div panics.
func (s Signed) Div(other Signed) Signed {
	return NewSigned(s.V.Div(other.V), s.Neg != other.Neg)
}

// DivMod calculates such quo and rem, that a = b * quo + rem. If b == 0, Div panics.
// See Value.DivMod() for more details.
func (s Signed) DivMod(other Signed, prec int) (quo, rem Signed) {
	q, r := s.V.DivMod(other.V, prec)
	neg := s.Neg != other.Neg
	return NewSigned(q, neg), NewSigned(r, s.Sign() < 0)
}

// Normalized normalizes signed value.
// See Value.Normalize().
func (s Signed) Normalized() Signed {
	if s.V.IsZero() {
		return signedZero
	}
	return NewSigned(s.V.Normalized(), s.Neg)
}

// Eq returns a==b.
func (s Signed) Eq(other Signed) bool {
	if s.Neg != other.Neg {
		return s.V.IsZero() && other.V.IsZero()
	}
	return s.V.Eq(other.V)
}

// Cmp compares two values.
// Returns -1 if a < b, 0 if a == b, 1 if a > b
func (s Signed) Cmp(other Signed) int {
	s1, s2 := s.Sign(), other.Sign()
	if s1 > s2 {
		return 1
	} else if s1 < s2 {
		return -1
	}
	return s.V.Cmp(other.V) * s1
}

// Sign returns -1 if a < 0, 0 if a = 0, 1 if a > 0.
func (s Signed) Sign() int {
	if s.V.IsZero() {
		return 0
	}
	if s.Neg {
		return -1
	}
	return 1
}

/*func (s Signed) Round(prec int) Signed {
	return NewSigned(s.V.Round(prec), s.Neg)
}*/

// AddValue returns a+v.
func (s Signed) AddValue(v Value) Signed {
	if !s.Neg {
		return PosValue(s.V.Add(v))
	}
	return NewSigned(v.Sub(s.V))
}

// SubValue returns a-v.
func (s Signed) SubValue(v Value) Signed {
	if s.Neg {
		return NegValue(s.V.Add(v))
	}
	return NewSigned(s.V.Sub(v))
}

// MulValue returns a*v.
func (s Signed) MulValue(v Value) Signed {
	return NewSigned(s.V.Mul(v), s.Neg)
}

// DivValue returns a/v.
func (s Signed) DivValue(v Value) Signed {
	return NewSigned(s.V.Div(v), s.Neg)
}

// DivModValue returns DivMod for a positive value.
func (s Signed) DivModValue(other Value, prec int) (quo, rem Signed) {
	return s.DivMod(PosValue(other), prec)
}

// EqValue returns a==v.
func (s Signed) EqValue(other Value) bool {
	if s.Neg {
		return s.V.IsZero() && other.IsZero()
	}
	return s.V.Eq(other)
}

// CmpValue compares signed and a positive values.
// Returns -1 if a < b, 0 if a == b, 1 if a > b
func (s Signed) CmpValue(other Value) int {
	if s.Neg {
		if s.V.IsZero() && other.IsZero() {
			return 0
		}
		return -1
	}
	return s.V.Cmp(other)
}
