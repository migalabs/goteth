package fork_block

import "github.com/attestantio/go-eth2-client/spec"

func NewPhase0Block(block spec.VersionedSignedBeaconBlock) ForkBlockContentBase {
	return ForkBlockContentBase{
		Slot:          uint64(block.Phase0.Message.Slot),
		ProposerIndex: uint64(block.Phase0.Message.ProposerIndex),
		Graffiti:      string(block.Phase0.Message.Body.Graffiti),
		Attestations:  block.Phase0.Message.Body.Attestations,
		Deposits:      block.Phase0.Message.Body.Deposits,
	}
}
