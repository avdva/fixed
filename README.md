# fixed [![GoDoc](https://godoc.org/github.com/avdva/fixed?status.svg)](http://godoc.org/github.com/avdva/fixed)
`fixed` implements a fixed-point number, where both mantissa and exponent are stored in a single number.
Can be used to represent currency rates with up to 16 digits of precision.

## Installation
`	$  go get github.com/avdva/fixed`

## Representation

The Value is a 64 bit unsigned integer, where 8 high bits are used for exponent 56 bits for mantissa.