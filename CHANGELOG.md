## 0.7.0 (TBA)

FEATURES:

* Project tree has been reorganised. Instead of a one top level pakage it now contains a `dfp` (decimal floating-point) package
and a `fixed` (fixed-point integer) package.
* Added `Floor`, `Round`, `Ceil`.
* Added `MustFromString`, `MustFromFloat64`.

IMPROVEMENTS:

* Make all tests pass for expBits = 7, 9.
* Update `github.com/stretchr/testify` to v1.5.1.

## 0.6.0 (April 10, 2020)

FIXES:

* Fixed possible incorrect result for FromFloat64.

FEATURES:

* value: add `Div` and `DivMod` to divide values.
* value: add `IsZero`.

IMPROVEMENTS:

* added mode tests for `FromFloat64`, `FromString`, JSON unmarshalling.
* readme: added goreportcard badge.

## 0.5.0 (March 29, 2020)

FEATURES:

* value: add `Mul` to multiply values.

## 0.4.0 (February 18, 2020)

FEATURES:

* value: add `Add` to sum values.

## 0.3.0 (February 09, 2020)

FEATURES:

* value: add `Cmp` and `Eq` to compare values.

IMPROVEMENTS:

* ci: add CircleCI support and a badge.
* readme: add more information.
* docs: add examples.

## 0.2.1 (January 28, 2020)

FEATURES:

* all: initial release.