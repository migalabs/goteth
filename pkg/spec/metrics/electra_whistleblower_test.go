package metrics

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"

	"github.com/migalabs/goteth/pkg/spec"
)

// From Electra on, slash_validator divides by WHISTLEBLOWER_REWARD_QUOTIENT_ELECTRA
// (4096), not the phase0 512. With the old quotient the proposer was credited
// eight times the whistleblower reward it receives on chain: at slot 15038043
// the proposer's balance rose by 58_302_762 gwei, which is what the beacon node
// reports, while the stored manual reward held 112_990_262.
//
// Two validators with different effective balances are slashed, so the
// division is pinned per validator and not only on the sum.
func electraWhistleblowerBundle(block *spec.AgnosticBlock) *ElectraMetrics {
	const epoch = phase0.Epoch(100)
	mkVals := func() []*phase0.Validator {
		vals := make([]*phase0.Validator, 4)
		for i := range vals {
			vals[i] = &phase0.Validator{
				EffectiveBalance:  phase0.Gwei(32_000_000_000),
				WithdrawableEpoch: phase0.Epoch(1) << 40, // slashable window still open
				ExitEpoch:         phase0.Epoch(spec.FarFutureEpoch),
			}
		}
		vals[0].EffectiveBalance = phase0.Gwei(2_048_000_000_000) // compounding, 2048 ETH
		return vals
	}
	block.Slot = phase0.Slot(uint64(epoch) * spec.SlotsPerEpoch)
	block.ProposerIndex = 2
	block.Proposed = true

	next := &spec.AgnosticState{StateRoot: phase0.Root{0x01}, Epoch: epoch, Validators: mkVals(), Blocks: []*spec.AgnosticBlock{block}}
	current := &spec.AgnosticState{StateRoot: phase0.Root{0x01}, Epoch: epoch - 1, Validators: mkVals()}
	m := &ElectraMetrics{}
	m.InitBundle(next, current, &spec.AgnosticState{})
	return m
}

// 2048 ETH / 4096 + 32 ETH / 4096. The proposer is also the whistleblower, so
// it keeps the whole of both.
const wantElectraWhistleblower = phase0.Gwei(500_000_000 + 7_812_500)

func TestElectraAttesterSlashingUsesTheElectraQuotient(t *testing.T) {
	block := &spec.AgnosticBlock{ElectraAttesterSlashings: []*electra.AttesterSlashing{{
		Attestation1: &electra.IndexedAttestation{AttestingIndices: []uint64{0, 1, 3}},
		Attestation2: &electra.IndexedAttestation{AttestingIndices: []uint64{0, 1}},
	}}}
	m := electraWhistleblowerBundle(block)

	m.ProcessSlashings()

	assert.Equal(t, wantElectraWhistleblower, block.ManualReward)
	assert.Equal(t, wantElectraWhistleblower, m.baseMetrics.MaxSlashingRewards[block.ProposerIndex])
	assert.Equal(t, 2, m.baseMetrics.NextState.NewAttesterSlashings)
}

func TestElectraProposerSlashingUsesTheElectraQuotient(t *testing.T) {
	header := func(idx phase0.ValidatorIndex) *phase0.SignedBeaconBlockHeader {
		return &phase0.SignedBeaconBlockHeader{Message: &phase0.BeaconBlockHeader{ProposerIndex: idx}}
	}
	block := &spec.AgnosticBlock{ProposerSlashings: []*phase0.ProposerSlashing{
		{SignedHeader1: header(0), SignedHeader2: header(0)},
		{SignedHeader1: header(1), SignedHeader2: header(1)},
	}}
	m := electraWhistleblowerBundle(block)

	m.ProcessSlashings()

	assert.Equal(t, wantElectraWhistleblower, block.ManualReward)
	assert.Equal(t, wantElectraWhistleblower, m.baseMetrics.MaxSlashingRewards[block.ProposerIndex])
	assert.Equal(t, 2, m.baseMetrics.NextState.NewProposerSlashings)
}
