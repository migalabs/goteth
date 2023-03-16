package fork_block

import (
	"github.com/attestantio/go-eth2-client/spec"
)

func NewCapellaBlock(block spec.VersionedSignedBeaconBlock) ForkBlockContentBase {
	return ForkBlockContentBase{
		Slot:              uint64(block.Capella.Message.Slot),
		ProposerIndex:     uint64(block.Capella.Message.ProposerIndex),
		Graffiti:          block.Capella.Message.Body.Graffiti,
		Attestations:      block.Capella.Message.Body.Attestations,
		Deposits:          block.Capella.Message.Body.Deposits,
		ProposerSlashings: block.Capella.Message.Body.ProposerSlashings,
		AttesterSlashings: block.Capella.Message.Body.AttesterSlashings,
		VoluntaryExits:    block.Capella.Message.Body.VoluntaryExits,
		SyncAggregate:     block.Capella.Message.Body.SyncAggregate,
		ExecutionPayload: ForkBlockPayloadBase{
			FeeRecipient:  block.Capella.Message.Body.ExecutionPayload.FeeRecipient,
			GasLimit:      block.Capella.Message.Body.ExecutionPayload.GasLimit,
			GasUsed:       block.Capella.Message.Body.ExecutionPayload.GasUsed,
			Timestamp:     block.Capella.Message.Body.ExecutionPayload.Timestamp,
			BaseFeePerGas: block.Capella.Message.Body.ExecutionPayload.BaseFeePerGas,
			BlockHash:     block.Capella.Message.Body.ExecutionPayload.BlockHash,
			Transactions:  block.Capella.Message.Body.ExecutionPayload.Transactions,
			BlockNumber:   block.Capella.Message.Body.ExecutionPayload.BlockNumber,
			Withdrawals:   block.Capella.Message.Body.ExecutionPayload.Withdrawals,
		},
	}
}
