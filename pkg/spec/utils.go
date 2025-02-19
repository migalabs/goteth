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

type BlockRewards struct {
	ExecutionOptimistic bool                `json:"execution_optimistic"`
	Finalized           bool                `json:"finalized"`
	Data                BlockRewardsContent `json:"data"`
}

type BlockRewardsContent struct {
	ProposerIndex     uint64 `json:"proposer_index,string"`
	Total             uint64 `json:"total,string"`
	Attestations      uint64 `json:"attestations,string"`
	SyncAggregate     uint64 `json:"sync_aggregate,string"`
	ProposerSlashings uint64 `json:"proposer_slashings,string"`
	AttesterSlashings uint64 `json:"attester_slashings,string"`
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
