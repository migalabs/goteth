package analyzer

import (
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/migalabs/goteth/pkg/spec"
)

const (
	// aggregationMinRewindEpochs is the smallest bound on how far back a restart may move its
	// first slot so that the first rewards aggregation window starts right after the last
	// persisted one. The usual distance is about one window plus epochsToFinalizedTentative.
	aggregationMinRewindEpochs = phase0.Epoch(64)
	// aggregationLookupAttempts is how many times the last persisted window is read before the
	// analyzer exits: starting a new window sequence would break t_validator_rewards_aggregation.
	aggregationLookupAttempts = 5
	aggregationLookupBackoff  = 5 * time.Second
)

// aggregationMaxRewind is the furthest a restart may move back to continue the window sequence.
func aggregationMaxRewind(windowEpochs int) phase0.Epoch {
	r := phase0.Epoch(2*windowEpochs + epochsToFinalizedTentative)
	if r < aggregationMinRewindEpochs {
		r = aggregationMinRewindEpochs
	}
	return r
}

// aggregationResume says where a (re)started analyzer downloads from and where its first rewards
// aggregation window starts.
type aggregationResume struct {
	NextSlot phase0.Slot  // first slot to download
	Start    phase0.Epoch // first epoch of the first window
	End      phase0.Epoch // last epoch of the first window, inclusive
	Rewound  bool         // NextSlot was moved back to cover the epochs right after the last window
	Gap      bool         // the epochs right after the last window are too far back: they stay uncovered
	Ignored  bool         // the persisted windows end implausibly far ahead of the chain head
}

// planAggregationResume keeps the rewards aggregation windows contiguous across restarts.
//
// nextSlot is the first slot the analyzer would download. Rewards are first produced for epoch
// EpochAtSlot(nextSlot)+2, which is where the first window used to start: every restart therefore
// began a new window sequence, leaving overlapping or missing epochs in
// t_validator_rewards_aggregation next to the last window of the previous run.
//
// nextWindowStart is the first epoch not covered by a persisted window (found is false for an
// empty table), headEpoch the current chain head.
//   - nextWindowStart later than the first rewards epoch: the epochs in between are processed
//     again but not aggregated.
//   - earlier: nextSlot moves back so they are processed and aggregated, up to
//     aggregationMaxRewind epochs. Further back, the window sequence keeps its grid and the
//     uncovered epochs are whole windows.
//   - more than one window beyond the head: the table holds windows this chain cannot have
//     produced yet, so they are ignored rather than holding back aggregation.
func planAggregationResume(nextSlot phase0.Slot, windowEpochs int, nextWindowStart phase0.Epoch, found bool, headEpoch phase0.Epoch) aggregationResume {
	nextSlot = spec.FirstSlotInEpoch(nextSlot)
	firstRewardsEpoch := spec.EpochAtSlot(nextSlot) + 2
	r := aggregationResume{NextSlot: nextSlot, Start: firstRewardsEpoch}

	if found && windowEpochs > 1 {
		n := phase0.Epoch(windowEpochs)
		switch {
		case nextWindowStart > headEpoch+n:
			r.Ignored = true
		case nextWindowStart >= firstRewardsEpoch:
			r.Start = nextWindowStart
		case nextWindowStart >= 2 && firstRewardsEpoch-nextWindowStart <= aggregationMaxRewind(windowEpochs):
			r.Start = nextWindowStart
			r.NextSlot = spec.ComputeStartSlotAtEpoch(nextWindowStart - 2)
			r.Rewound = true
		default:
			// Keep the grid of the persisted windows: the first window at or after the first
			// rewards epoch, so the missing epochs are whole windows that can be backfilled.
			r.Start = nextWindowStart + (firstRewardsEpoch-nextWindowStart+n-1)/n*n
			r.Gap = true
		}
	}

	if windowEpochs > 1 {
		r.End = r.Start + phase0.Epoch(windowEpochs-1)
	} else {
		r.End = r.Start
	}
	return r
}

// planRewardsAggregationResume reads where the persisted windows end and applies
// planAggregationResume. If the last window cannot be read after a few attempts the analyzer
// exits, since a new window sequence would break the table and a restart is safe.
func (s *ChainAnalyzer) planRewardsAggregationResume(nextSlot phase0.Slot, headSlot phase0.Slot) aggregationResume {
	var next phase0.Epoch
	found := false
	if s.rewardsAggregationEpochs > 1 {
		var err error
		for attempt := 1; ; attempt++ {
			next, found, err = s.dbClient.RetrieveNextAggregationStart()
			if err == nil {
				break
			}
			if attempt >= aggregationLookupAttempts {
				log.Fatalf("could not read the last rewards aggregation window after %d attempts: %s", attempt, err)
			}
			log.Warnf("could not read the last rewards aggregation window (attempt %d): %s", attempt, err)
			time.Sleep(aggregationLookupBackoff)
		}
	}
	r := planAggregationResume(nextSlot, s.rewardsAggregationEpochs, next, found, spec.EpochAtSlot(headSlot))
	firstRewardsEpoch := spec.EpochAtSlot(spec.FirstSlotInEpoch(nextSlot)) + 2
	switch {
	case r.Ignored:
		log.Warnf("rewards aggregation: the persisted windows end at epoch %d, beyond the head epoch %d; starting a new window sequence at epoch %d",
			next-1, spec.EpochAtSlot(headSlot), r.Start)
	case r.Rewound:
		log.Infof("rewards aggregation continues at epoch %d: download moved back from slot %d to slot %d",
			r.Start, spec.FirstSlotInEpoch(nextSlot), r.NextSlot)
	case r.Gap:
		log.Warnf("rewards aggregation: epochs %d-%d are too far back to aggregate, the window sequence continues at epoch %d",
			next, r.Start-1, r.Start)
	case found && r.Start > firstRewardsEpoch:
		log.Infof("rewards aggregation continues at epoch %d: epochs %d-%d are processed again without aggregating them",
			r.Start, firstRewardsEpoch, r.Start-1)
	case found:
		log.Infof("rewards aggregation continues at epoch %d", r.Start)
	}
	return r
}
