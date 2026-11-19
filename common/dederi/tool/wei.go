package tool

import (
	"github.com/shopspring/decimal"
	"math/big"
)

func DecimalToWei(number decimal.Decimal) *big.Int {
	return number.Mul(decimal.NewFromInt(1e18)).BigInt()
}

func WeiToDecimal(number *big.Int) decimal.Decimal {
	return decimal.NewFromBigInt(number, -18)
}
