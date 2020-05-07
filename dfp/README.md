# Package dfp

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
