package spec

import (
	"math"

	api "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type EpochDuties struct {
	ProposerDuties   []*api.ProposerDuty                   // 32 Proposer Duties per Epoch
	BeaconCommittees []*api.BeaconCommittee                // Beacon Committees organized by slot for the whole epoch
	ValidatorAttSlot map[phase0.ValidatorIndex]phase0.Slot // for each validator we have which slot it had to attest to
}

func (p EpochDuties) GetValList(slot uint64, committeeIndex uint64) []phase0.ValidatorIndex {
	for _, committee := range p.BeaconCommittees {
		if (uint64(committee.Slot) == slot) && (uint64(committee.Index) == committeeIndex) {
			return committee.Validators
		}
	}

	return nil
}

func GetEffectiveBalance(balance float64) float64 {
	return math.Min(MaxEffectiveInc*EffectiveBalanceInc, balance)
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
