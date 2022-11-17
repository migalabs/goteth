package fork_block

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.WithField(
		"module", "FoskBlockContent",
	)
)

// This Wrapper is meant to include all common objects across Ethereum Hard Fork Specs
type ForkBlockContentBase struct {
	Slot          uint64
	ProposerIndex uint64
	Graffiti      string

	Attestations []*phase0.Attestation
	Deposits     []*phase0.Deposit
}

func GetCustomBlock(block spec.VersionedSignedBeaconBlock, iApi *http.Service) (ForkBlockContentBase, error) {
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

	return ForkBlockContentBase{}, nil
}
