package spec

import (
	"regexp"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.WithField(
		"module", "spec",
	)
)

// https://github.com/ethereum/consensus-specs/blob/master/specs/phase0/beacon-chain.md#compute_start_slot_at_epoch
func ComputeStartSlotAtEpoch(epoch phase0.Epoch) phase0.Slot {
	return phase0.Slot(uint64(epoch) * SlotsPerEpoch)
}

func FirstSlotInEpoch(slot phase0.Slot) phase0.Slot {
	return slot / SlotsPerEpoch * SlotsPerEpoch
}

func EpochAtSlot(slot phase0.Slot) phase0.Epoch {
	return phase0.Epoch(slot / SlotsPerEpoch)
}

func HexStringAddressIsValid(address string) bool {
	hexPattern := regexp.MustCompile(`^(0x)?[0-9a-fA-F]+$`)
	return len(address) == 42 && hexPattern.MatchString(address)
}

func Uint64Max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func Uint64Min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
