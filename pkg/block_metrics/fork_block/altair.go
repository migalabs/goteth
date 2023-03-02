package fork_block

import (
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func NewAltairBlock(block spec.VersionedSignedBeaconBlock) ForkBlockContentBase {
	return ForkBlockContentBase{
		Slot:              uint64(block.Altair.Message.Slot),
		ProposerIndex:     uint64(block.Altair.Message.ProposerIndex),
		Graffiti:          block.Altair.Message.Body.Graffiti,
		Attestations:      block.Altair.Message.Body.Attestations,
		Deposits:          block.Altair.Message.Body.Deposits,
		ProposerSlashings: block.Altair.Message.Body.ProposerSlashings,
		AttesterSlashings: block.Altair.Message.Body.AttesterSlashings,
		VoluntaryExits:    block.Altair.Message.Body.VoluntaryExits,
		SyncAggregate:     block.Altair.Message.Body.SyncAggregate,
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
