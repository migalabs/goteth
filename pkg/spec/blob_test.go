package spec_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
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

// blobTx encodes a type-3 transaction committing the given blob hashes, in the
// form a block's execution payload carries, and returns its hash.
//
// The transaction is unsigned, so the hash is not one that would appear on
// chain - a signature changes it. That does not matter here: what attribution
// needs is that each hash is derived from its own transaction rather than
// chosen by the test, and that two transactions differ. The nonce provides the
// second part, so two transactions committing identical blobs still hash
// differently, which is the situation that produced the bug.
func blobTx(t *testing.T, nonce uint64, blobSeeds ...byte) (bellatrix.Transaction, common.Hash) {
	t.Helper()

	hashes := make([]common.Hash, 0, len(blobSeeds))
	for _, seed := range blobSeeds {
		hashes = append(hashes, blobHash(seed))
	}
	tx := types.NewTx(&types.BlobTx{
		ChainID:    uint256.NewInt(1),
		Nonce:      nonce,
		GasTipCap:  uint256.NewInt(1),
		GasFeeCap:  uint256.NewInt(1),
		Gas:        21000,
		Value:      uint256.NewInt(0),
		BlobFeeCap: uint256.NewInt(1),
		BlobHashes: hashes,
	})
	raw, err := tx.MarshalBinary()
	if err != nil {
		t.Fatalf("encoding blob transaction: %s", err)
	}
	return bellatrix.Transaction(raw), tx.Hash()
}

// plainTx encodes a transaction that carries no blobs.
func plainTx(t *testing.T, nonce uint64) bellatrix.Transaction {
	t.Helper()

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   uint256.NewInt(1).ToBig(),
		Nonce:     nonce,
		GasTipCap: uint256.NewInt(1).ToBig(),
		GasFeeCap: uint256.NewInt(1).ToBig(),
		Gas:       21000,
		Value:     uint256.NewInt(0).ToBig(),
	})
	raw, err := tx.MarshalBinary()
	if err != nil {
		t.Fatalf("encoding plain transaction: %s", err)
	}
	return bellatrix.Transaction(raw)
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

// expect pairs each sidecar, in the order they were handed over, with the
// transaction hash it should end up carrying.
func expect(blobs []*spec.AgnosticBlobSidecar, hashes ...common.Hash) []string {
	want := make([]string, 0, len(blobs))
	for i, b := range blobs {
		want = append(want, fmt.Sprintf("%d:%s", b.Index, hashes[i].String()))
	}
	return want
}

// unattributed is the expectation when nothing should be assigned at all.
func unattributed(blobs []*spec.AgnosticBlobSidecar) []string {
	want := make([]string, 0, len(blobs))
	for _, b := range blobs {
		want = append(want, fmt.Sprintf("%d:%s", b.Index, common.Hash{}.String()))
	}
	return want
}

func TestAssignTxHashes(t *testing.T) {
	tests := []struct {
		name    string
		build   func(t *testing.T) ([]*spec.AgnosticBlobSidecar, []bellatrix.Transaction, []string)
		wantErr bool
	}{
		{
			// Mainnet slot 14414626. Three consecutive transactions each carried
			// the same near-empty blob, so all three shared one versioned hash.
			// Matching on that hash collapsed every copy onto the last of them.
			name: "transactions carrying identical blobs keep their own",
			build: func(t *testing.T) ([]*spec.AgnosticBlobSidecar, []bellatrix.Transaction, []string) {
				t0, h0 := blobTx(t, 0, 1, 2, 3, 4, 5)
				t1, h1 := blobTx(t, 1, 9)
				t2, h2 := blobTx(t, 2, 9)
				t3, h3 := blobTx(t, 3, 9)
				t4, h4 := blobTx(t, 4, 6)
				t5, h5 := blobTx(t, 5, 7)

				blobs := sidecars(1, 2, 3, 4, 5, 9, 9, 9, 6, 7)
				txs := []bellatrix.Transaction{t0, t1, t2, t3, t4, t5}
				want := expect(blobs, h0, h0, h0, h0, h0, h1, h2, h3, h4, h5)
				return blobs, txs, want
			},
		},
		{
			// The review case: a blob transaction whose EL receipt has not
			// arrived. Attribution reads the block body, so the receipt is
			// irrelevant and every blob in the slot is still attributed.
			name: "a blob transaction whose receipt never arrived",
			build: func(t *testing.T) ([]*spec.AgnosticBlobSidecar, []bellatrix.Transaction, []string) {
				t0, h0 := blobTx(t, 0, 1)
				t1, h1 := blobTx(t, 1, 2)
				t2, h2 := blobTx(t, 2, 3)

				blobs := sidecars(1, 2, 3)
				// All three are in the block. Only one produced a receipt, so
				// AgnosticTransactions would have held a single entry.
				txs := []bellatrix.Transaction{t0, t1, t2}
				return blobs, txs, expect(blobs, h0, h1, h2)
			},
		},
		{
			name: "one transaction carrying the same blob twice",
			build: func(t *testing.T) ([]*spec.AgnosticBlobSidecar, []bellatrix.Transaction, []string) {
				tx, h := blobTx(t, 0, 4, 4)
				blobs := sidecars(4, 4)
				return blobs, []bellatrix.Transaction{tx}, expect(blobs, h, h)
			},
		},
		{
			name: "transactions without blobs are ignored",
			build: func(t *testing.T) ([]*spec.AgnosticBlobSidecar, []bellatrix.Transaction, []string) {
				t1, h1 := blobTx(t, 1, 1)
				t3, h3 := blobTx(t, 3, 2)
				blobs := sidecars(1, 2)
				txs := []bellatrix.Transaction{plainTx(t, 0), t1, plainTx(t, 2), t3}
				return blobs, txs, expect(blobs, h1, h3)
			},
		},
		{
			// Sidecars need not arrive in index order; the index decides, not
			// the position in the slice.
			name: "sidecars given out of order",
			build: func(t *testing.T) ([]*spec.AgnosticBlobSidecar, []bellatrix.Transaction, []string) {
				t0, h0 := blobTx(t, 0, 1, 2)
				t1, h1 := blobTx(t, 1, 3)
				blobs := []*spec.AgnosticBlobSidecar{
					{Index: 2, BlobHash: blobHash(3).String()},
					{Index: 0, BlobHash: blobHash(1).String()},
					{Index: 1, BlobHash: blobHash(2).String()},
				}
				return blobs, []bellatrix.Transaction{t0, t1}, expect(blobs, h1, h0, h0)
			},
		},
		{
			// More sidecars than the block committed to. Attributing them
			// positionally would shift every blob onto the wrong transaction.
			name: "more sidecars than the block committed",
			build: func(t *testing.T) ([]*spec.AgnosticBlobSidecar, []bellatrix.Transaction, []string) {
				t0, _ := blobTx(t, 0, 1)
				blobs := sidecars(1, 2, 3)
				return blobs, []bellatrix.Transaction{t0}, unattributed(blobs)
			},
			wantErr: true,
		},
		{
			// The other direction, and the one the index-bound check cannot
			// see: every sidecar present is valid and in range, but the block
			// committed a blob that never arrived. Attributing the ones that
			// did would publish a slot that is quietly short a blob.
			name: "fewer sidecars than the block committed",
			build: func(t *testing.T) ([]*spec.AgnosticBlobSidecar, []bellatrix.Transaction, []string) {
				t0, _ := blobTx(t, 0, 1, 2, 3)
				blobs := sidecars(1, 2)
				return blobs, []bellatrix.Transaction{t0}, unattributed(blobs)
			},
			wantErr: true,
		},
		{
			name: "a sidecar claiming an index past the block's blobs",
			build: func(t *testing.T) ([]*spec.AgnosticBlobSidecar, []bellatrix.Transaction, []string) {
				t0, _ := blobTx(t, 0, 1)
				t1, _ := blobTx(t, 1, 2)
				blobs := []*spec.AgnosticBlobSidecar{
					{Index: 0, BlobHash: blobHash(1).String()},
					{Index: 7, BlobHash: blobHash(2).String()},
				}
				return blobs, []bellatrix.Transaction{t0, t1}, unattributed(blobs)
			},
			wantErr: true,
		},
		{
			// Index is a uint64 from the beacon API. Narrowing one at or above
			// 2^63 to int wraps it negative, which passes a "too large" check
			// and then panics on the bounds array, taking block processing with
			// it.
			name: "a sidecar with an index that wraps when narrowed",
			build: func(t *testing.T) ([]*spec.AgnosticBlobSidecar, []bellatrix.Transaction, []string) {
				t0, _ := blobTx(t, 0, 1)
				blobs := []*spec.AgnosticBlobSidecar{
					{Index: deneb.BlobIndex(1 << 63), BlobHash: blobHash(1).String()},
				}
				return blobs, []bellatrix.Transaction{t0}, unattributed(blobs)
			},
			wantErr: true,
		},
		{
			name: "two sidecars claiming the same index",
			build: func(t *testing.T) ([]*spec.AgnosticBlobSidecar, []bellatrix.Transaction, []string) {
				t0, _ := blobTx(t, 0, 1)
				t1, _ := blobTx(t, 1, 1)
				blobs := []*spec.AgnosticBlobSidecar{
					{Index: 0, BlobHash: blobHash(1).String()},
					{Index: 0, BlobHash: blobHash(1).String()},
				}
				return blobs, []bellatrix.Transaction{t0, t1}, unattributed(blobs)
			},
			wantErr: true,
		},
		{
			name: "sidecars that the transactions did not commit to",
			build: func(t *testing.T) ([]*spec.AgnosticBlobSidecar, []bellatrix.Transaction, []string) {
				t0, _ := blobTx(t, 0, 1)
				t1, _ := blobTx(t, 1, 2)
				blobs := sidecars(1, 8)
				return blobs, []bellatrix.Transaction{t0, t1}, unattributed(blobs)
			},
			wantErr: true,
		},
		{
			// A payload entry that is not a valid transaction is a decoding
			// failure, not a mismatch, and must not be silently skipped.
			name: "an undecodable transaction",
			build: func(t *testing.T) ([]*spec.AgnosticBlobSidecar, []bellatrix.Transaction, []string) {
				blobs := sidecars(1)
				return blobs, []bellatrix.Transaction{{0xff, 0xff, 0xff}}, unattributed(blobs)
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			blobs, txs, want := test.build(t)

			err := spec.AssignTxHashes(blobs, txs)

			if test.wantErr && err == nil {
				t.Fatal("expected an error, got none")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			got := attributed(blobs)
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("blob %d: got %s, want %s", i, got[i], want[i])
				}
			}
		})
	}
}

func TestAssignTxHashesCountsOnlyBlobCarriersInItsError(t *testing.T) {
	// The error is the only thing anyone sees when a slot goes unattributed, so
	// the transaction count in it has to mean "transactions that committed
	// blobs", not "transactions in the block". Counting plain transactions here
	// would send whoever reads the log looking in the wrong place.
	t1, _ := blobTx(t, 1, 1)
	txs := []bellatrix.Transaction{plainTx(t, 0), t1, plainTx(t, 2), plainTx(t, 3)}

	err := spec.AssignTxHashes(sidecars(1, 2), txs)
	if err == nil {
		t.Fatal("expected a mismatch error, got none")
	}
	if !strings.Contains(err.Error(), "by 1 transactions") {
		t.Errorf("error should report 1 blob-carrying transaction, got: %s", err)
	}
}

func TestAssignTxHashesReportsBothHashesOnMismatch(t *testing.T) {
	// The error is all anyone gets when a slot goes unattributed. Naming only
	// the index leaves them unable to tell which sidecar landed where it should
	// not have, which matters most in exactly the case this function exists
	// for: several blobs in one block sharing a versioned hash.
	tx, _ := blobTx(t, 0, 1)
	arrived, committed := blobHash(8).String(), blobHash(1).String()

	err := spec.AssignTxHashes(
		[]*spec.AgnosticBlobSidecar{{Index: 0, BlobHash: arrived}},
		[]bellatrix.Transaction{tx},
	)
	if err == nil {
		t.Fatal("expected a hash mismatch error, got none")
	}
	msg := err.Error()
	gotAt, wantAt := strings.Index(msg, arrived), strings.Index(msg, committed)
	if gotAt < 0 {
		t.Errorf("error omits the hash that arrived (%s): %s", arrived, msg)
	}
	if wantAt < 0 {
		t.Errorf("error omits the hash the block committed (%s): %s", committed, msg)
	}
	// Naming both but the wrong way round is worse than naming neither: it
	// reads plausibly and sends the reader looking for the wrong sidecar. The
	// hash that arrived is described first, the one the block committed second.
	if gotAt >= 0 && wantAt >= 0 && gotAt > wantAt {
		t.Errorf("the hashes are the wrong way round; the arrived hash should come first: %s", msg)
	}
}
