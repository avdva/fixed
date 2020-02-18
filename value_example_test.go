// Copyright 2020 Aleksandr Demakin. All rights reserved.

package fixed

import (
	"encoding/json"
	"fmt"
)

func ExampleValue() {
	v1, err := FromString("1.23456")
	if err != nil {
		panic(err)
	}
	fmt.Printf("v1 as a float = %v, mantissa = %v, uint64 = %v\n", v1.Float64(), v1.MantUint64(), v1.Uint64())

	v2, err := FromFloat64(1.23456)
	if err != nil {
		panic(err)
	}
	fmt.Printf("value from string: %s, value from float: %s, values are equal: %v\n", v1.String(), v2.String(), v1.Eq(v2))

	v3, err := FromMantAndExp(12345, -4)
	if err != nil {
		panic(err)
	}
	fmt.Printf("uint64 values for -6 exp %d, %d\n", v1.ToExp(-6).MantUint64(), v3.ToExp(-6).MantUint64())

	data, err := json.Marshal(v1)
	if err != nil {
		panic(err)
	}
	fmt.Printf("json for value: %s\n", string(data))

	JSONMode = JSONModeME
	data, err = json.Marshal(v1)
	if err != nil {
		panic(err)
	}
	fmt.Printf("json for value and JSONModeME: %s\n", string(data))

	v4, err := FromString("1234560")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s + %s = %s", v4.String(), v1.String(), v4.Add(v1).String())

	// Output:
	// v1 as a float = 1.23456, mantissa = 123456, uint64 = 1
	// value from string: 1.23456, value from float: 1.23456, values are equal: true
	// uint64 values for -6 exp 1234560, 1234500
	// json for value: "1.23456"
	// json for value and JSONModeME: {"m":123456,"e":-5}
	// 1234560 + 1.23456 = 1234561.23456
}
