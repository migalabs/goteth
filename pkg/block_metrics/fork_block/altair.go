package fork_block

import (
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/model"
)

func NewAltairBlock(block spec.VersionedSignedBeaconBlock) model.ForkBlockContentBase {
	return model.ForkBlockContentBase{
		Slot:              block.Altair.Message.Slot,
		ProposerIndex:     block.Altair.Message.ProposerIndex,
		Graffiti:          block.Altair.Message.Body.Graffiti,
		Proposed:          true,
		Attestations:      block.Altair.Message.Body.Attestations,
		Deposits:          block.Altair.Message.Body.Deposits,
		ProposerSlashings: block.Altair.Message.Body.ProposerSlashings,
		AttesterSlashings: block.Altair.Message.Body.AttesterSlashings,
		VoluntaryExits:    block.Altair.Message.Body.VoluntaryExits,
		SyncAggregate:     block.Altair.Message.Body.SyncAggregate,
		ExecutionPayload: model.ForkBlockPayloadBase{
			FeeRecipient:  bellatrix.ExecutionAddress{},
			GasLimit:      0,
			GasUsed:       0,
			Timestamp:     0,
			BaseFeePerGas: [32]byte{},
			BlockHash:     phase0.Hash32{},
			Transactions:  make([]bellatrix.Transaction, 0),
			BlockNumber:   0,
			Withdrawals:   make([]*capella.Withdrawal, 0),
		},
	}
}
