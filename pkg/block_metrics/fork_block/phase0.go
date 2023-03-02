package fork_block

import (
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func NewPhase0Block(block spec.VersionedSignedBeaconBlock) ForkBlockContentBase {
	return ForkBlockContentBase{
		Slot:              uint64(block.Phase0.Message.Slot),
		ProposerIndex:     uint64(block.Phase0.Message.ProposerIndex),
		Graffiti:          block.Phase0.Message.Body.Graffiti,
		Attestations:      block.Phase0.Message.Body.Attestations,
		Deposits:          block.Phase0.Message.Body.Deposits,
		ProposerSlashings: block.Phase0.Message.Body.ProposerSlashings,
		AttesterSlashings: block.Phase0.Message.Body.AttesterSlashings,
		VoluntaryExits:    block.Phase0.Message.Body.VoluntaryExits,
		SyncAggregate:     &altair.SyncAggregate{},
		ExecutionPayload: ForkBlockPayloadBase{
			FeeRecipient:  bellatrix.ExecutionAddress{},
			GasLimit:      0,
			GasUsed:       0,
			Timestamp:     0,
			BaseFeePerGas: [32]byte{},
			BlockHash:     phase0.Hash32{},
			Transactions:  make([]bellatrix.Transaction, 0),
			BlockNumber:   0,
		},
	}
}
