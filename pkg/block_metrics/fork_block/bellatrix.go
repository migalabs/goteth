package fork_block

import (
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/model"
)

func NewBellatrixBlock(block spec.VersionedSignedBeaconBlock) model.ForkBlockContentBase {
	return model.ForkBlockContentBase{
		Slot:              block.Bellatrix.Message.Slot,
		ProposerIndex:     block.Bellatrix.Message.ProposerIndex,
		Graffiti:          block.Bellatrix.Message.Body.Graffiti,
		Proposed:          true,
		Attestations:      block.Bellatrix.Message.Body.Attestations,
		Deposits:          block.Bellatrix.Message.Body.Deposits,
		ProposerSlashings: block.Bellatrix.Message.Body.ProposerSlashings,
		AttesterSlashings: block.Bellatrix.Message.Body.AttesterSlashings,
		VoluntaryExits:    block.Bellatrix.Message.Body.VoluntaryExits,
		SyncAggregate:     block.Bellatrix.Message.Body.SyncAggregate,
		ExecutionPayload: model.ForkBlockPayloadBase{
			FeeRecipient:  block.Bellatrix.Message.Body.ExecutionPayload.FeeRecipient,
			GasLimit:      block.Bellatrix.Message.Body.ExecutionPayload.GasLimit,
			GasUsed:       block.Bellatrix.Message.Body.ExecutionPayload.GasUsed,
			Timestamp:     block.Bellatrix.Message.Body.ExecutionPayload.Timestamp,
			BaseFeePerGas: block.Bellatrix.Message.Body.ExecutionPayload.BaseFeePerGas,
			BlockHash:     block.Bellatrix.Message.Body.ExecutionPayload.BlockHash,
			Transactions:  block.Bellatrix.Message.Body.ExecutionPayload.Transactions,
			BlockNumber:   block.Bellatrix.Message.Body.ExecutionPayload.BlockNumber,
			Withdrawals:   make([]*capella.Withdrawal, 0),
		},
	}
}
