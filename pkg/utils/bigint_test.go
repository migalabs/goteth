package utils_test

import (
	"math/big"
	"testing"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/migalabs/goteth/pkg/utils"
)

func TestBigIntToUInt256(t *testing.T) {
	mustBig := func(s string) *big.Int {
		v, ok := new(big.Int).SetString(s, 10)
		if !ok {
			t.Fatalf("invalid bigint: %s", s)
		}
		return v
	}

	tests := []struct {
		name string
		in   *big.Int
		want proto.UInt256
	}{
		{
			name: "nil",
			in:   nil,
			want: proto.UInt256{},
		},
		{
			name: "negative",
			in:   big.NewInt(-1),
			want: proto.UInt256{},
		},
		{
			name: "zero",
			in:   big.NewInt(0),
			want: proto.UInt256{},
		},
		{
			name: "one",
			in:   big.NewInt(1),
			want: proto.UInt256{Low: proto.UInt128{Low: 1}},
		},
		{
			name: "max uint64",
			in:   new(big.Int).SetUint64(^uint64(0)),
			want: proto.UInt256{Low: proto.UInt128{Low: ^uint64(0)}},
		},
		{
			name: "2^64",
			in:   new(big.Int).Lsh(big.NewInt(1), 64),
			want: proto.UInt256{Low: proto.UInt128{High: 1}},
		},
		{
			name: "2^128",
			in:   new(big.Int).Lsh(big.NewInt(1), 128),
			want: proto.UInt256{High: proto.UInt128{Low: 1}},
		},
		{
			name: "2^192",
			in:   new(big.Int).Lsh(big.NewInt(1), 192),
			want: proto.UInt256{High: proto.UInt128{High: 1}},
		},
		{
			name: "max uint256",
			in:   new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)),
			want: proto.UInt256{
				Low:  proto.UInt128{Low: ^uint64(0), High: ^uint64(0)},
				High: proto.UInt128{Low: ^uint64(0), High: ^uint64(0)},
			},
		},
		{
			name: "overflow (2^256)",
			in:   new(big.Int).Lsh(big.NewInt(1), 256),
			want: proto.UInt256{},
		},
		{
			name: "100 ETH in wei",
			in:   mustBig("100000000000000000000"),
			want: proto.UInt256{Low: proto.UInt128{Low: 7766279631452241920, High: 5}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.BigIntToUInt256(tt.in)
			if got != tt.want {
				t.Errorf("BigIntToUInt256(%v) = %+v, want %+v", tt.in, got, tt.want)
			}
		})
	}
}
