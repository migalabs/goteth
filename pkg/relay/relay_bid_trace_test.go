package relay

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/migalabs/goteth/pkg/spec"
	"github.com/stretchr/testify/assert"
)

// https://github.com/ethereum/consensus-specs/blob/dev/specs/deneb/beacon-chain.md#kzg_commitment_to_versioned_hash
func TestRelayBids(t *testing.T) {

	cli, err := InitRelaysMonitorer(context.Background(), spec.MainnetGenesis)
	if err != nil {
		return
	}

	bidTraces, err := cli.GetDeliveredBidsPerSlotRange(8639732, 1)

	if err != nil {
		return
	}

	for _, slotBids := range bidTraces.bids {
		for _, relay := range slotBids {
			fmt.Print(relay.BuilderPubkey.String())
			assert.Equal(t, "0x8dde59a0d40b9a77b901fc40bee1116acf643b2b60656ace951a5073fe317f57a086acf1eac7502ea32edcca1a900521", relay.BuilderPubkey.String())
			assert.Equal(t, "0xe7a77627659c62fd9a8e7984e0558e134efb02d10c11313836915ed9190cbfdd", relay.BlockHash.String())
			assert.Equal(t, uint64(21411944), relay.GasUsed)
			assert.Equal(t, *big.NewInt(61630396424352492), *relay.Value)
		}

	}
}
