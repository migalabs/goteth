package spec

import (
	"bytes"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func IsCorrectSource(attestation phase0.Attestation, block AgnosticBlock) bool {
	return block.Slot-attestation.Data.Slot <= 5
}

func IsCorrectTarget(attestation phase0.Attestation, firstBlockInEpoch AgnosticBlock) bool {
	attEpoch := int(attestation.Data.Slot / 32)
	firstSlotOfEpoch := phase0.Slot(attEpoch * 32)

	if firstSlotOfEpoch == firstBlockInEpoch.Slot {
		// provided block is at the good slot
		return bytes.Equal(firstBlockInEpoch.Root[:], attestation.Data.Target.Root[:])
	}

	return false

}

func IsCorrectHead(attestation phase0.Attestation, block AgnosticBlock) bool {
	if bytes.Equal(block.ParentRoot[:], attestation.Data.BeaconBlockRoot[:]) {
		if block.Slot-attestation.Data.Slot == 1 {
			return true
		}
	}
	return false
}
