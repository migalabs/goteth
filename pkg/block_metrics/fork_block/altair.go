package fork_block

import (
	"github.com/attestantio/go-eth2-client/spec"
)

func NewAltairBlock(block spec.VersionedSignedBeaconBlock) ForkBlockContentBase {
	return ForkBlockContentBase{
		Slot:          uint64(block.Altair.Message.Slot),
		ProposerIndex: uint64(block.Altair.Message.ProposerIndex),
		Graffiti:      string(block.Altair.Message.Body.Graffiti),
		Attestations:  block.Altair.Message.Body.Attestations,
		Deposits:      block.Altair.Message.Body.Deposits,
	}
}
