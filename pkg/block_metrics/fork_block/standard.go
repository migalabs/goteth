package fork_block

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/model"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.WithField(
		"module", "ForkBlockContent",
	)
)

func GetCustomBlock(block spec.VersionedSignedBeaconBlock) (model.ForkBlockContentBase, error) {
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
		return model.ForkBlockContentBase{}, fmt.Errorf("could not figure out the Beacon Block Fork Version: %s", block.Version)
	}
}
