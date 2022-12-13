package fork_block

import "github.com/attestantio/go-eth2-client/spec"

func NewBellatrixBlock(block spec.VersionedSignedBeaconBlock) ForkBlockContentBase {
	return ForkBlockContentBase{
		Slot:          uint64(block.Bellatrix.Message.Slot),
		ProposerIndex: uint64(block.Bellatrix.Message.ProposerIndex),
		Graffiti:      block.Bellatrix.Message.Body.Graffiti,
		Attestations:  block.Bellatrix.Message.Body.Attestations,
		Deposits:      block.Bellatrix.Message.Body.Deposits,
	}
}
