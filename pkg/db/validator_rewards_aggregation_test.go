package db

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func TestNextAggregationStart(t *testing.T) {
	tests := []struct {
		name                                   string
		lastStart, lastEnd, lastRows, prevRows uint64
		want                                   phase0.Epoch
	}{
		{name: "complete window", lastStart: 492066, lastEnd: 492075, lastRows: 765029, prevRows: 765232, want: 492076},
		{name: "only one window", lastStart: 100, lastEnd: 109, lastRows: 5, prevRows: 0, want: 110},
		{name: "fewer validators within tolerance", lastStart: 100, lastEnd: 109, lastRows: 990, prevRows: 1000, want: 110},
		{name: "partially persisted window", lastStart: 100, lastEnd: 109, lastRows: 400000, prevRows: 765000, want: 100},
		{name: "more validators", lastStart: 100, lastEnd: 109, lastRows: 1200, prevRows: 1000, want: 110},
		{name: "trimmed window keeps its own end", lastStart: 100, lastEnd: 104, lastRows: 1000, prevRows: 1000, want: 105},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := nextAggregationStart(tc.lastStart, tc.lastEnd, tc.lastRows, tc.prevRows); got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}
