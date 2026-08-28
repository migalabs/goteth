package spec_test

import (
	"fmt"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/migalabs/goteth/pkg/spec"
)

// blobHash builds a distinguishable versioned hash. Passing the same seed twice
// produces the same hash, which is what happens on chain when two transactions
// carry byte-identical blobs.
func blobHash(seed byte) common.Hash {
	var h common.Hash
	h[0] = 0x01
	h[31] = seed
	return h
}

func carrier(txIdx uint64, txSeed byte, blobSeeds ...byte) spec.AgnosticTransaction {
	var txHash phase0.Hash32
	txHash[31] = txSeed

	hashes := make([]common.Hash, 0, len(blobSeeds))
	for _, seed := range blobSeeds {
		hashes = append(hashes, blobHash(seed))
	}
	return spec.AgnosticTransaction{TxIdx: txIdx, Hash: txHash, BlobHashes: hashes}
}

// sidecars builds the block's sidecar list, indexed in block order.
func sidecars(blobSeeds ...byte) []*spec.AgnosticBlobSidecar {
	blobs := make([]*spec.AgnosticBlobSidecar, 0, len(blobSeeds))
	for i, seed := range blobSeeds {
		blobs = append(blobs, &spec.AgnosticBlobSidecar{
			Index:    deneb.BlobIndex(i),
			BlobHash: blobHash(seed).String(),
		})
	}
	return blobs
}

func attributed(blobs []*spec.AgnosticBlobSidecar) []string {
	got := make([]string, 0, len(blobs))
	for _, b := range blobs {
		got = append(got, fmt.Sprintf("%d:%s", b.Index, b.TxHash.String()))
	}
	return got
}

func expectedAttribution(pairs ...uint64) []string {
	want := make([]string, 0, len(pairs))
	for i := 0; i < len(pairs); i += 2 {
		var txHash phase0.Hash32
		txHash[31] = byte(pairs[i+1])
		want = append(want, fmt.Sprintf("%d:%s", pairs[i], common.Hash(txHash).String()))
	}
	return want
}

func TestAssignTxHashes(t *testing.T) {
	tests := []struct {
		name    string
		blobs   []*spec.AgnosticBlobSidecar
		txs     []spec.AgnosticTransaction
		want    []string
		wantErr bool
	}{
		{
			// Mainnet slot 14414626. Three consecutive transactions each carried
			// the same near-empty blob, so all three shared one versioned hash.
			// Matching on that hash collapsed every copy onto transaction 132.
			name:  "transactions carrying identical blobs keep their own",
			blobs: sidecars(1, 2, 3, 4, 5, 9, 9, 9, 6, 7),
			txs: []spec.AgnosticTransaction{
				carrier(9, 0xaa, 1, 2, 3, 4, 5),
				carrier(130, 0xbb, 9),
				carrier(131, 0xcc, 9),
				carrier(132, 0xdd, 9),
				carrier(201, 0xee, 6),
				carrier(471, 0xff, 7),
			},
			want: expectedAttribution(
				0, 0xaa, 1, 0xaa, 2, 0xaa, 3, 0xaa, 4, 0xaa,
				5, 0xbb, 6, 0xcc, 7, 0xdd,
				8, 0xee, 9, 0xff),
		},
		{
			name:  "one transaction carrying the same blob twice",
			blobs: sidecars(4, 4),
			txs:   []spec.AgnosticTransaction{carrier(3, 0xaa, 4, 4)},
			want:  expectedAttribution(0, 0xaa, 1, 0xaa),
		},
		{
			name:  "transactions without blobs are ignored",
			blobs: sidecars(1, 2),
			txs: []spec.AgnosticTransaction{
				{TxIdx: 0},
				carrier(1, 0xaa, 1),
				{TxIdx: 2},
				carrier(3, 0xbb, 2),
			},
			want: expectedAttribution(0, 0xaa, 1, 0xbb),
		},
		{
			// A transaction whose receipt never arrived is dropped while parsing,
			// so the sidecars outnumber the blobs the transactions account for.
			// Attributing them positionally would shift every blob onto the wrong
			// transaction, so nothing is attributed at all.
			name:  "a missing transaction leaves the mapping untouched",
			blobs: sidecars(1, 2, 3),
			txs: []spec.AgnosticTransaction{
				carrier(1, 0xaa, 1),
				carrier(9, 0xbb, 3),
			},
			want:    expectedAttribution(0, 0, 1, 0, 2, 0),
			wantErr: true,
		},
		{
			// Nothing else in the block tells these apart, so an index the
			// block never committed has to be refused rather than folded onto
			// whichever transaction happens to sit nearby.
			name: "a sidecar claiming an index past the block's blobs",
			blobs: []*spec.AgnosticBlobSidecar{
				{Index: 0, BlobHash: blobHash(1).String()},
				{Index: 7, BlobHash: blobHash(2).String()},
			},
			txs: []spec.AgnosticTransaction{
				carrier(1, 0xaa, 1),
				carrier(2, 0xbb, 2),
			},
			want:    expectedAttribution(0, 0, 7, 0),
			wantErr: true,
		},
		{
			name: "two sidecars claiming the same index",
			blobs: []*spec.AgnosticBlobSidecar{
				{Index: 0, BlobHash: blobHash(1).String()},
				{Index: 0, BlobHash: blobHash(1).String()},
			},
			txs: []spec.AgnosticTransaction{
				carrier(1, 0xaa, 1),
				carrier(2, 0xbb, 1),
			},
			want:    expectedAttribution(0, 0, 0, 0),
			wantErr: true,
		},
		{
			// Sidecars need not arrive in index order.
			name: "sidecars given out of order",
			blobs: []*spec.AgnosticBlobSidecar{
				{Index: 2, BlobHash: blobHash(3).String()},
				{Index: 0, BlobHash: blobHash(1).String()},
				{Index: 1, BlobHash: blobHash(2).String()},
			},
			txs: []spec.AgnosticTransaction{
				carrier(1, 0xaa, 1, 2),
				carrier(2, 0xbb, 3),
			},
			want: expectedAttribution(2, 0xbb, 0, 0xaa, 1, 0xaa),
		},
		{
			// Transaction order decides the mapping, not the order the caller
			// happened to hand them over in.
			name:  "transactions given out of order",
			blobs: sidecars(1, 2, 3),
			txs: []spec.AgnosticTransaction{
				carrier(9, 0xbb, 3),
				carrier(1, 0xaa, 1, 2),
			},
			want: expectedAttribution(0, 0xaa, 1, 0xaa, 2, 0xbb),
		},
		{
			name:  "sidecars that the transactions did not commit to",
			blobs: sidecars(1, 8),
			txs: []spec.AgnosticTransaction{
				carrier(1, 0xaa, 1),
				carrier(2, 0xbb, 2),
			},
			want:    expectedAttribution(0, 0, 1, 0),
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := spec.AssignTxHashes(test.blobs, test.txs)

			if test.wantErr && err == nil {
				t.Fatal("expected an error, got none")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			got := attributed(test.blobs)
			for i := range test.want {
				if got[i] != test.want[i] {
					t.Errorf("blob %d: got %s, want %s", i, got[i], test.want[i])
				}
			}
		})
	}
}
