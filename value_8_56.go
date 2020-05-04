package fixed

const (
	signBit     = 0
	expBits     = 8
	bias        = -(1<<(expBits-1) - 1)
	maxExponent = 1 << (expBits - 1)
	minExponent = bias

	expMask  = 1<<expBits - 1
	mantMask = 1<<mantBits - 1
)

var (
	zero = Value(0)
)

func exp(v Value) expType {
	return expType(v>>mantBits&expMask) + bias
}

func mant(v Value) number {
	return number(v & mantMask)
}

func split(v Value) (mantissa number, exponent expType) {
	return mant(v), exp(v)
}

func fromMantAndExp(mant number, exp expType) Value {
	return Value(number(exp-bias)<<mantBits | (mant & mantMask))
}
