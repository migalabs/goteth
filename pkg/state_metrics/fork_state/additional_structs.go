package fork_state

import (
	"context"
	"math"
	"strconv"

	api "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
)

type EpochData struct {
	ProposerDuties   []*api.ProposerDuty                   // 32 Proposer Duties per Epoch
	BeaconCommittees []*api.BeaconCommittee                // Beacon Committees organized by slot for the whole epoch
	ValidatorAttSlot map[phase0.ValidatorIndex]phase0.Slot // for each validator we have which slot it had to attest to
}

func NewEpochData(iApi *http.Service, slot uint64) EpochData {

	epochCommittees, err := iApi.BeaconCommittees(context.Background(), strconv.Itoa(int(slot)))

	if err != nil {
		log.Errorf(err.Error())
	}

	validatorsAttSlot := make(map[phase0.ValidatorIndex]phase0.Slot) // each validator, when it had to attest
	validatorsPerSlot := make(map[phase0.Slot][]phase0.ValidatorIndex)

	for _, committee := range epochCommittees {
		for _, valID := range committee.Validators {
			validatorsAttSlot[valID] = committee.Slot

			if val, ok := validatorsPerSlot[committee.Slot]; ok {
				// the slot exists in the map
				validatorsPerSlot[committee.Slot] = append(val, valID)
			} else {
				// the slot does not exist, create
				validatorsPerSlot[committee.Slot] = []phase0.ValidatorIndex{valID}
			}
		}
	}

	proposerDuties, err := iApi.ProposerDuties(context.Background(), phase0.Epoch(utils.GetEpochFromSlot(uint64(slot))), nil)

	if err != nil {
		log.Errorf(err.Error())
	}

	return EpochData{
		ProposerDuties:   proposerDuties,
		BeaconCommittees: epochCommittees,
		ValidatorAttSlot: validatorsAttSlot,
	}
}

func (p EpochData) GetValList(slot uint64, committeeIndex uint64) []phase0.ValidatorIndex {
	for _, committee := range p.BeaconCommittees {
		if (uint64(committee.Slot) == slot) && (uint64(committee.Index) == committeeIndex) {
			return committee.Validators
		}
	}

	return nil
}

func GetEffectiveBalance(balance float64) float64 {
	return math.Min(MAX_EFFECTIVE_INCREMENTS*EFFECTIVE_BALANCE_INCREMENT, balance)
}

type ValVote struct {
	ValId         uint64
	AttestedSlot  []uint64
	InclusionSlot []uint64
}

func (p *ValVote) AddNewAtt(attestedSlot uint64, inclusionSlot uint64) {

	if p.AttestedSlot == nil {
		p.AttestedSlot = make([]uint64, 0)
	}

	if p.InclusionSlot == nil {
		p.InclusionSlot = make([]uint64, 0)
	}

	// keep in mind that for the proposer, the vote only counts if it is the first to include this attestation
	for i, item := range p.AttestedSlot {
		if item == attestedSlot {
			if inclusionSlot < p.InclusionSlot[i] {
				p.InclusionSlot[i] = inclusionSlot
			}
			return
		}
	}

	p.AttestedSlot = append(p.AttestedSlot, attestedSlot)
	p.InclusionSlot = append(p.InclusionSlot, inclusionSlot)

}

func GweiToUint64(iArray []phase0.Gwei) []uint64 {
	result := make([]uint64, 0)

	for _, item := range iArray {
		result = append(result, uint64(item))
	}
	return result
}

func RootToByte(iArray []phase0.Root) [][]byte {
	result := make([][]byte, len(iArray))

	for i, item := range iArray {
		result[i] = make([]byte, len(item))
		copy(result[i], item[:])
	}
	return result
}
