package fork_state

import (
	"math"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

var ( // spec weight constants
	TIMELY_SOURCE_WEIGHT       = 14
	TIMELY_TARGET_WEIGHT       = 26
	TIMELY_HEAD_WEIGHT         = 14
	PARTICIPATING_FLAGS_WEIGHT = []int{TIMELY_SOURCE_WEIGHT, TIMELY_TARGET_WEIGHT, TIMELY_HEAD_WEIGHT}
	SYNC_REWARD_WEIGHT         = 2
	PROPOSER_WEIGHT            = 8
	WEIGHT_DENOMINATOR         = 64
	SYNC_COMMITTEE_SIZE        = 512
)

// This Wrapper is meant to include all necessary data from the Altair Fork
func NewAltairState(bstate spec.VersionedBeaconState, iApi *http.Service) ForkStateContentBase {

	altairObj := ForkStateContentBase{
		Version:       bstate.Version,
		Balances:      bstate.Altair.Balances,
		Validators:    bstate.Altair.Validators,
		EpochStructs:  NewEpochData(iApi, uint64(bstate.Altair.Slot)),
		Epoch:         phase0.Epoch(bstate.Altair.Slot / SLOTS_PER_EPOCH),
		Slot:          bstate.Altair.Slot,
		BlockRoots:    RootToByte(bstate.Altair.BlockRoots),
		SyncCommittee: *bstate.Altair.CurrentSyncCommittee,
	}

	altairObj.Setup()

	ProcessAttestations(&altairObj, bstate.Altair.PreviousEpochParticipation)

	return altairObj
}

func ProcessAttestations(customState *ForkStateContentBase, participation []altair.ParticipationFlags) {
	// calculate attesting vals only once
	flags := []altair.ParticipationFlag{
		altair.TimelySourceFlagIndex,
		altair.TimelyTargetFlagIndex,
		altair.TimelyHeadFlagIndex}

	for participatingFlag := range flags {

		flag := altair.ParticipationFlags(math.Pow(2, float64(participatingFlag)))

		for valIndex, item := range participation {
			// Here we have one item per validator
			// Item is a 3-bit string
			// each bit represents a flag

			if (item & flag) == flag {
				// The attestation has a timely flag, therefore we consider it correct flag
				customState.CorrectFlags[participatingFlag][valIndex] += uint(1)

				// we sum the attesting balance in the corresponding flag index
				customState.AttestingBalance[participatingFlag] += customState.Validators[valIndex].EffectiveBalance

				// if this validator was not counted as attesting before, count it now
				if !customState.AttestingVals[valIndex] {
					customState.NumAttestingVals++
					customState.MaxAttestingBalance = customState.Validators[valIndex].EffectiveBalance
				}
				customState.AttestingVals[valIndex] = true
			}
		}
	}
}
