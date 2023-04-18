package model

import (
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

// This Wrapper is meant to include all common objects across Ethereum Hard Fork Specs
type ForkBlockContentBase struct {
	Slot              phase0.Slot
	ProposerIndex     phase0.ValidatorIndex
	Graffiti          [32]byte
	Proposed          bool
	Attestations      []*phase0.Attestation
	Deposits          []*phase0.Deposit
	ProposerSlashings []*phase0.ProposerSlashing
	AttesterSlashings []*phase0.AttesterSlashing
	VoluntaryExits    []*phase0.SignedVoluntaryExit
	SyncAggregate     *altair.SyncAggregate
	ExecutionPayload  ForkBlockPayloadBase
}

// This Wrapper is meant to include all common objects across Ethereum Hard Fork Specs
type ForkBlockPayloadBase struct {
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

func (f ForkBlockContentBase) InsertOp() bool {
	return true
}

func (p ForkBlockPayloadBase) BaseFeeToInt() int {
	return 0 // not implemented yet
}
