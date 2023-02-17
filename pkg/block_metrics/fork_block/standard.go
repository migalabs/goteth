package fork_block

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.WithField(
		"module", "ForkBlockContent",
	)
)

// This Wrapper is meant to include all common objects across Ethereum Hard Fork Specs
type ForkBlockContentBase struct {
	Slot          uint64
	ProposerIndex uint64
	Graffiti      [32]byte

	Attestations      []*phase0.Attestation
	Deposits          []*phase0.Deposit
	ProposerSlashings []*phase0.ProposerSlashing
	AttesterSlashings []*phase0.AttesterSlashing
	VoluntaryExits    []*phase0.SignedVoluntaryExit
	SyncAggregate     *altair.SyncAggregate
	ExecutionPayload  ForkBlockPayloadBase
}

func GetCustomBlock(block spec.VersionedSignedBeaconBlock) (ForkBlockContentBase, error) {
	switch block.Version {

	case spec.DataVersionPhase0:
		return NewPhase0Block(block), nil

	case spec.DataVersionAltair:
		return NewAltairBlock(block), nil

	case spec.DataVersionBellatrix:
		return NewBellatrixBlock(block), nil
	default:
		return ForkBlockContentBase{}, fmt.Errorf("could not figure out the Beacon Block Fork Version: %s", block.Version)
	}
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
}
