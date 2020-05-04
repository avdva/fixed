// Copyright 2020 Aleksandr Demakin. All rights reserved.

package fixed

import (
	"fmt"
)

type priceSize struct {
	Price, Size Value
}

func ExampleSigned() {
	obook := []priceSize{
		{MustFromFloat64(1.2345), MustFromFloat64(3.3)},
		{MustFromFloat64(1.235), MustFromFloat64(1.4)},
		{MustFromFloat64(1.2357), MustFromFloat64(4)},
		{MustFromFloat64(1.23571), MustFromFloat64(2.5)},
		{MustFromFloat64(1.23582), MustFromFloat64(1.5)},
	}

	vw, vol := vwap(obook, MustFromFloat64(10))
	fmt.Printf("vwap for orber book is %s with volume %s\n", vw.Round(5), vol)

	vw, vol = vwap(obook, MustFromFloat64(15))
	fmt.Printf("vwap for orber book is %s with volume %s\n", vw.Round(5), vol)

	// Output:
	// vwap for orber book is 1.23521 with volume 10
	// vwap for orber book is 1.23533 with volume 12.7
}

func vwap(obook []priceSize, desiredVolume Value) (vwap, vol Value) {
	var tier, spent Signed
	v := PosValue(desiredVolume)
	for _, it := range obook {
		left := v.Sub(tier)
		if left.Sign() <= 0 {
			break
		}
		sz := PosValue(it.Size)
		if left.Cmp(sz) < 0 {
			sz = left
		}
		tier = tier.Add(sz)
		spent = spent.Add(sz.Mul(PosValue(it.Price)))
	}
	return spent.Div(tier).V, tier.V
}
