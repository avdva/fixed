package dfp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

const (
	delim = '.'
)

var (
	manyZeros = bytes.Repeat([]byte{'0'}, 256)
)

type posError struct {
	pos int
	err string
}

func newPosError(err string, pos int) *posError {
	return &posError{err: err, pos: pos}
}

func (pe posError) Error() string {
	return pe.err + fmt.Sprintf(" at pos %d", pe.pos)
}

func addPosErrorOffset(err error, offset int) error {
	var pe *posError
	if !errors.As(err, &pe) { // try to locate error position.
		return err
	}
	pe.pos += offset
	return pe
}

func parse(s string) (digits string, e int32, neg bool, err error) {
	s, offset, neg := prepareString(s)
	if len(s) == 0 {
		return "", 0, false, fmt.Errorf("empty input")
	}
	digits, e, err = doParse(s)
	if err != nil {
		// add what we've trimmed before and add +1 to the offset to start indices from 1.
		err = fmt.Errorf("parsing failed: %w", addPosErrorOffset(err, offset+1))
	}
	return digits, e, neg, err
}

// doParse parses given decimal string.
// returns a string without leading and trailing zeros, and an exponent
func doParse(s string) (result string, e int32, err error) {
	result, delimPos, e, err := removeLeadingZeros(s)
	if err != nil {
		return "", 0, err
	}
	result, eFromDelim := removeTrailingZerosString(result, delimPos)
	return result, e + eFromDelim, nil
}

// prepareString cleans the string from ",-,+ symbols, and spaces.
func prepareString(s string) (prepared string, offset int, neg bool) {
	if len(s) == 0 {
		return "", 0, false
	}
	if s[0] == '"' {
		s = s[1:]
		offset++
	}
	if len(s) == 0 {
		return "", 0, false
	}
	if s[len(s)-1] == '"' {
		s = s[:len(s)-1]
	}
	if trimmed := strings.TrimLeftFunc(s, unicode.IsSpace); len(trimmed) != len(s) {
		offset += len(s) - len(trimmed)
		s = trimmed
	}
	s = strings.TrimRightFunc(s, unicode.IsSpace)
	if len(s) == 0 {
		return "", 0, false
	}
	if s[0] == '-' {
		neg = true
		offset++
		s = s[1:]
	} else if s[0] == '+' {
		offset++
		s = s[1:]
	}
	return s, offset, neg
}

func removeLeadingZeros(s string) (result string, delimPos int, e int32, err error) {
	var b strings.Builder
	delimPos, firstNonZeroPos := -1, -1
outer:
	for i, r := range s {
		switch {
		case '0' <= r && r <= '9':
			if b.Len() == 0 {
				if r == '0' { // trim leading zeros
					continue
				}
				firstNonZeroPos = i
			}
			b.WriteRune(r)
		case r == 'e':
			parsed, err := strconv.ParseInt(s[i+1:], 10, 64)
			if err != nil {
				return "", 0, 0, newPosError("error parsing exponent: "+err.Error(), i+1)
			}
			e = int32(parsed)
			break outer
		case r == delim:
			if delimPos != -1 {
				return "", 0, 0, newPosError("unexpected delimeter", i)
			}
			delimPos = i
		default:
			return "", 0, 0, newPosError(fmt.Sprintf("unexpected symbol %q", r), i)
		}
	}
	if firstNonZeroPos == -1 { // a zero-only string
		return "", 0, 0, nil
	}

	result = b.String()

	// move delimPos to the beginning of the trimmed string
	if delimPos >= 0 {
		if delimPos < firstNonZeroPos {
			firstNonZeroPos--
		}
		delimPos -= firstNonZeroPos
	} else { // if there is no delim, add one at the end of the string 123 --> 123.
		delimPos = len(result)
	}

	return result, delimPos, e, nil
}

func removeTrailingZerosString(s string, delimPos int) (result string, e int32) {
	for {
		l := len(s)
		if l == 0 || s[l-1] != '0' {
			break
		}
		s = s[:l-1]
	}
	return s, int32(delimPos - len(s))
}

func formatMantExp(sign int, mant uint64, exp int32, format rune, w io.Writer) error {
	switch format {
	case 'f', 's':
		formatAsDecimal(mant, sign, exp, w)
	case 'e', 'v':
		fallthrough
	default:
		formatWithExponent(mant, sign, exp, w)
	}
	return nil
}

func formatAsDecimal(mant uint64, sign int, exp int32, w io.Writer) {
	if mant == 0 {
		w.Write([]byte{'0'})
		return
	}
	if sign < 0 {
		w.Write([]byte{'-'})
	}
	mString := strconv.FormatUint(mant, 10)
	switch {
	case exp >= 0:
		w.Write([]byte(mString))
		if exp > 0 {
			zeros := zeroBytes(int(exp))
			w.Write(zeros)
		}
	default:
		if diff := len(mString) + int(exp); diff <= 0 { // add leading zeros and a delimiter
			zeros := zeroBytes(-diff)
			w.Write([]byte{'0', delim})
			w.Write(zeros)
			w.Write([]byte(mString))
		} else { // insert a delimeter
			w.Write([]byte(mString[:diff]))
			w.Write([]byte{delim})
			w.Write([]byte(mString[diff:]))
		}
	}
}

func formatWithExponent(mant uint64, sign int, exp int32, w io.Writer) {
	if sign < 0 {
		w.Write([]byte{'-'})
	}
	mString := strconv.FormatUint(mant, 10)
	w.Write([]byte(mString))
	if mant != 0 {
		w.Write([]byte("e" + strconv.FormatInt(int64(exp), 10)))
	}
}

func zeroStr(count int) string {
	var b bytes.Buffer
	for i := 0; i < count/len(manyZeros); i++ {
		b.Write(manyZeros)
	}
	if rem := count % len(manyZeros); rem > 0 {
		b.Write(manyZeros[:rem])
	}
	return b.String()
}

func zeroBytes(count int) []byte {
	if count <= len(manyZeros) {
		return manyZeros[:count]
	}
	result := bytes.Repeat(manyZeros, count/len(manyZeros))
	if rem := count % len(manyZeros); rem > 0 {
		result = append(result, manyZeros[:rem]...)
	}
	return result
}
