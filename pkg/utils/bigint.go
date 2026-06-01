package utils

import (
	"math/big"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/holiman/uint256"
)

// BigIntToUInt256 converts a non-negative *big.Int to a ClickHouse proto.UInt256.
// Returns the zero value for nil or out-of-range inputs.
func BigIntToUInt256(v *big.Int) proto.UInt256 {
	if v == nil || v.Sign() < 0 {
		return proto.UInt256{}
	}
	u, overflow := uint256.FromBig(v)
	if overflow {
		return proto.UInt256{}
	}
	return proto.UInt256{
		Low:  proto.UInt128{Low: u[0], High: u[1]},
		High: proto.UInt128{Low: u[2], High: u[3]},
	}
}
