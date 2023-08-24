package spec

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

// This Wrapper is meant to include all common objects across Ethereum Hard Fork Specs
type AgnosticBlock struct {
	Slot              phase0.Slot
	StateRoot         phase0.Root
	ProposerIndex     phase0.ValidatorIndex
	Graffiti          [32]byte
	Proposed          bool
	Attestations      []*phase0.Attestation
	Deposits          []*phase0.Deposit
	ProposerSlashings []*phase0.ProposerSlashing
	AttesterSlashings []*phase0.AttesterSlashing
	VoluntaryExits    []*phase0.SignedVoluntaryExit
	SyncAggregate     *altair.SyncAggregate
	ExecutionPayload  AgnosticExecutionPayload
	Size              uint32
	Reward            BlockRewards
}

// This Wrapper is meant to include all common objects across Ethereum Hard Fork Specs
type AgnosticExecutionPayload struct {
	FeeRecipient  bellatrix.ExecutionAddress
	GasLimit      uint64
	GasUsed       uint64
	Timestamp     uint64
	BaseFeePerGas [32]byte
	BlockHash     phase0.Hash32
	Transactions  []bellatrix.Transaction
	BlockNumber   uint64
	Withdrawals   []*capella.Withdrawal
}

func (f AgnosticBlock) Type() ModelType {
	return BlockModel
}

func (p AgnosticExecutionPayload) BaseFeeToInt() int {
	return 0 // not implemented yet
}

func GetCustomBlock(block spec.VersionedSignedBeaconBlock) (AgnosticBlock, error) {
	switch block.Version {

	case spec.DataVersionPhase0:
		return NewPhase0Block(block), nil

	case spec.DataVersionAltair:
		return NewAltairBlock(block), nil

	case spec.DataVersionBellatrix:
		return NewBellatrixBlock(block), nil
	case spec.DataVersionCapella:
		return NewCapellaBlock(block), nil
	default:
		return AgnosticBlock{}, fmt.Errorf("could not figure out the Beacon Block Fork Version: %s", block.Version)
	}
}

func NewPhase0Block(block spec.VersionedSignedBeaconBlock) AgnosticBlock {
	return AgnosticBlock{
		Slot:              block.Phase0.Message.Slot,
		ProposerIndex:     block.Phase0.Message.ProposerIndex,
		Graffiti:          block.Phase0.Message.Body.Graffiti,
		Proposed:          true,
		Attestations:      block.Phase0.Message.Body.Attestations,
		Deposits:          block.Phase0.Message.Body.Deposits,
		ProposerSlashings: block.Phase0.Message.Body.ProposerSlashings,
		AttesterSlashings: block.Phase0.Message.Body.AttesterSlashings,
		VoluntaryExits:    block.Phase0.Message.Body.VoluntaryExits,
		SyncAggregate:     &altair.SyncAggregate{},
		ExecutionPayload: AgnosticExecutionPayload{
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

func NewAltairBlock(block spec.VersionedSignedBeaconBlock) AgnosticBlock {
	return AgnosticBlock{
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
		ExecutionPayload: AgnosticExecutionPayload{
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

func NewBellatrixBlock(block spec.VersionedSignedBeaconBlock) AgnosticBlock {
	return AgnosticBlock{
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
		ExecutionPayload: AgnosticExecutionPayload{
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

func NewCapellaBlock(block spec.VersionedSignedBeaconBlock) AgnosticBlock {
	return AgnosticBlock{
		Slot:              block.Capella.Message.Slot,
		ProposerIndex:     block.Capella.Message.ProposerIndex,
		Graffiti:          block.Capella.Message.Body.Graffiti,
		Proposed:          true,
		Attestations:      block.Capella.Message.Body.Attestations,
		Deposits:          block.Capella.Message.Body.Deposits,
		ProposerSlashings: block.Capella.Message.Body.ProposerSlashings,
		AttesterSlashings: block.Capella.Message.Body.AttesterSlashings,
		VoluntaryExits:    block.Capella.Message.Body.VoluntaryExits,
		SyncAggregate:     block.Capella.Message.Body.SyncAggregate,
		ExecutionPayload: AgnosticExecutionPayload{
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
