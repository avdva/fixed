# numeric [![GoDoc](https://godoc.org/github.com/avdva/numeric?status.svg)](http://godoc.org/github.com/avdva/numeric) [![CircleCI Build Status](https://circleci.com/gh/avdva/numeric.svg?style=shield)](https://circleci.com/gh/avdva/numeric) [![Go Report Card](https://goreportcard.com/badge/github.com/avdva/numeric)](https://goreportcard.com/report/github.com/avdva/numeric) 
`numeric` implements a fixed-point number, where both mantissa and exponent are stored in a single number.
Can be used to represent currency rates with up to 16 digits of precision.

## Installation
`	$  go get github.com/avdva/numeric/dfp`

## Representation

Value is a positive decimal floating-point number.
It currently uses a uint64 value as a data type, where
8 bits are used for exponent and 56 for mantissa.

```
   63      55                                                     0
   ________|_______________________________________________________
   eeeeeeeemmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmm
```

Value can be useful for representing numbers like prices in financial services.
Negative numbers aren't currently supported.

## Usage

To get a value use one of `From{Uint64, String, Float, MantAndExp} `:

```
	v, err := dfp.FromString("1.23456")
	if err != nil {
		panic(err)
	}
```

See `value_example_test.go` for more examples.
