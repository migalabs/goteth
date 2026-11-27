package analyzer

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/migalabs/goteth/pkg/spec"
)

func slotOf(epoch phase0.Epoch) phase0.Slot { return spec.ComputeStartSlotAtEpoch(epoch) }

func TestPlanAggregationResume(t *testing.T) {
	tests := []struct {
		name            string
		nextSlot        phase0.Slot
		windowEpochs    int
		nextWindowStart phase0.Epoch
		found           bool
		head            phase0.Epoch // 0 means far ahead
		want            aggregationResume
	}{
		{
			// No persisted window: same as before, the first window starts at the first rewards epoch.
			name: "empty table", nextSlot: slotOf(100), windowEpochs: 10,
			want: aggregationResume{NextSlot: slotOf(100), Start: 102, End: 111},
		},
		{
			name: "aggregation disabled", nextSlot: slotOf(100), windowEpochs: 1, nextWindowStart: 95, found: true,
			want: aggregationResume{NextSlot: slotOf(100), Start: 102, End: 102},
		},
		{
			// Mainnet restart of 2026-11-26: resumed at slot 15740608 (epoch 491894, 4 epochs before
			// finalized), so the old code started a window at 491896 over the persisted 491890-491899.
			name: "restart before the last window end", nextSlot: slotOf(491894), windowEpochs: 10, nextWindowStart: 491900, found: true,
			want: aggregationResume{NextSlot: slotOf(491894), Start: 491900, End: 491909},
		},
		{
			name: "restart exactly at the next window", nextSlot: slotOf(98), windowEpochs: 10, nextWindowStart: 100, found: true,
			want: aggregationResume{NextSlot: slotOf(98), Start: 100, End: 109},
		},
		{
			// Hoodi: last window 138246-138255, the old code resumed with a window at 138258.
			name: "restart after the last window end", nextSlot: slotOf(138256), windowEpochs: 10, nextWindowStart: 138256, found: true,
			want: aggregationResume{NextSlot: slotOf(138254), Start: 138256, End: 138265, Rewound: true},
		},
		{
			name: "rewind at the limit", nextSlot: slotOf(1000), windowEpochs: 10, nextWindowStart: 1002 - 64, found: true,
			want: aggregationResume{NextSlot: slotOf(1000 - 64), Start: 1002 - 64, End: 1011 - 64, Rewound: true},
		},
		{
			// 65 epochs back: the sequence keeps its grid (937, 947, ...) at the first window after 1002.
			name: "too far back keeps the grid", nextSlot: slotOf(1000), windowEpochs: 10, nextWindowStart: 937, found: true,
			want: aggregationResume{NextSlot: slotOf(1000), Start: 1007, End: 1016, Gap: true},
		},
		{
			name: "too far back on the grid", nextSlot: slotOf(1000), windowEpochs: 10, nextWindowStart: 932, found: true,
			want: aggregationResume{NextSlot: slotOf(1000), Start: 1002, End: 1011, Gap: true},
		},
		{
			name: "window start below epoch 2", nextSlot: slotOf(10), windowEpochs: 10, nextWindowStart: 1, found: true,
			want: aggregationResume{NextSlot: slotOf(10), Start: 21, End: 30, Gap: true},
		},
		{
			// The bound grows with the window: 2*60+4 = 124 epochs.
			name: "large windows rewind further", nextSlot: slotOf(1000), windowEpochs: 60, nextWindowStart: 900, found: true,
			want: aggregationResume{NextSlot: slotOf(898), Start: 900, End: 959, Rewound: true},
		},
		{
			name: "windows beyond the head are ignored", nextSlot: slotOf(990), windowEpochs: 10, nextWindowStart: 1011, found: true, head: 1000,
			want: aggregationResume{NextSlot: slotOf(990), Start: 992, End: 1001, Ignored: true},
		},
		{
			name: "next window within one window of the head", nextSlot: slotOf(990), windowEpochs: 10, nextWindowStart: 1010, found: true, head: 1000,
			want: aggregationResume{NextSlot: slotOf(990), Start: 1010, End: 1019},
		},
		{
			name: "next slot not at an epoch start", nextSlot: slotOf(491894) + 5, windowEpochs: 10, nextWindowStart: 491900, found: true,
			want: aggregationResume{NextSlot: slotOf(491894), Start: 491900, End: 491909},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			head := tc.head
			if head == 0 {
				head = 1 << 40
			}
			got := planAggregationResume(tc.nextSlot, tc.windowEpochs, tc.nextWindowStart, tc.found, head)
			if got != tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

// The first window after a restart must start right after the last persisted one whenever the
// distance allows it, and processing must start early enough to produce its first epoch.
func TestPlanAggregationResumeIsContiguous(t *testing.T) {
	const windowEpochs = 10
	for nextWindowStart := phase0.Epoch(400); nextWindowStart <= 600; nextWindowStart++ {
		for resumeEpoch := phase0.Epoch(380); resumeEpoch <= 620; resumeEpoch++ {
			r := planAggregationResume(slotOf(resumeEpoch), windowEpochs, nextWindowStart, true, 1<<40)
			firstRewardsEpoch := spec.EpochAtSlot(r.NextSlot) + 2
			if r.Gap {
				if resumeEpoch+2 <= nextWindowStart+aggregationMaxRewind(windowEpochs) {
					t.Fatalf("resume %d, next window %d: unexpected gap", resumeEpoch, nextWindowStart)
				}
				if (r.Start-nextWindowStart)%windowEpochs != 0 || r.Start < firstRewardsEpoch || r.Start >= firstRewardsEpoch+windowEpochs {
					t.Fatalf("resume %d, next window %d: gap window %d-%d is off the grid", resumeEpoch, nextWindowStart, r.Start, r.End)
				}
				continue
			}
			if r.Start != nextWindowStart || r.End != nextWindowStart+windowEpochs-1 {
				t.Fatalf("resume %d, next window %d: window %d-%d", resumeEpoch, nextWindowStart, r.Start, r.End)
			}
			if firstRewardsEpoch > r.Start {
				t.Fatalf("resume %d, next window %d: rewards start at %d, after the window start %d", resumeEpoch, nextWindowStart, firstRewardsEpoch, r.Start)
			}
			if r.NextSlot > slotOf(resumeEpoch) {
				t.Fatalf("resume %d: the download start moved forward to slot %d", resumeEpoch, r.NextSlot)
			}
		}
	}
}
