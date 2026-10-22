package spec

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	api "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/migalabs/goteth/pkg/utils"
)

var (
	versionedHashVersionKZG = []byte("0x01")
)

type AgnosticBlobSidecar struct {
	Slot                        phase0.Slot // slot the blob belongs to
	TxHash                      common.Hash // has of the transactions that references this blob in this slot
	BlobHash                    string      // versioned blob hash
	Blob                        deneb.Blob  // the blob itself
	BlobEnding0s                int         // amount of consecutive 0s at the end of the blob
	Index                       deneb.BlobIndex
	KZGCommitment               deneb.KZGCommitment
	KZGProof                    deneb.KZGProof
	SignedBlockHeader           *phase0.SignedBeaconBlockHeader
	KZGCommitmentInclusionProof deneb.KZGCommitmentInclusionProof
}

func NewAgnosticBlobFromAPI(slot phase0.Slot, blob deneb.BlobSidecar) (*AgnosticBlobSidecar, error) {

	return &AgnosticBlobSidecar{
		Slot:                        slot,
		Index:                       blob.Index,
		Blob:                        blob.Blob,
		KZGCommitment:               blob.KZGCommitment,
		BlobHash:                    KZGCommitmentToVersionedHash(blob.KZGCommitment),
		KZGProof:                    blob.KZGProof,
		SignedBlockHeader:           blob.SignedBlockHeader,
		KZGCommitmentInclusionProof: blob.KZGCommitmentInclusionProof,
		BlobEnding0s:                utils.CountConsecutiveEnding0(blob.Blob[:]),
	}, nil
}

// AssignTxHashes maps every blob sidecar of a block to the transaction that
// carried it.
//
// A block's blob commitments are the concatenation of each type-3 transaction's
// blob hashes taken in transaction order, so the sidecar holding index i belongs
// to the transaction that owns the i-th entry of that concatenation.
//
// Position is the only dependable key. A versioned hash is derived from nothing
// but the blob's KZG commitment, so two transactions carrying byte-identical
// blobs share one hash and cannot be told apart by it; matching on the hash
// awards every copy to whichever of them is compared last, and leaves the
// others with no blobs at all.
//
// It reads the block's own transaction list rather than the parsed
// AgnosticTransactions, and that is deliberate. Everything needed here - each
// type-3 transaction's hash and the blob hashes it committed - is in the block
// body. AgnosticTransactions are built by matching transactions to EL receipts
// and only include the ones that matched, so a receipt that has not arrived
// drops a blob carrier from the list. Attribution would then see fewer blobs
// committed than sidecars delivered and refuse the whole slot, leaving every
// blob in it with no transaction. Partial receipt fetches happen often enough
// to have their own recovery path (recoverBlockReceipts), so this must not
// depend on them.
//
// Nothing is assigned until the whole block has been checked: the sidecars must
// account for exactly the blobs the transactions committed to, each index must
// appear once, and every hash must agree with the one committed at that
// position. Because the input no longer depends on receipts, a disagreement now
// means the sidecars are genuinely not the ones this block committed to, and
// leaving the mapping untouched is the right answer rather than a symptom of a
// slow EL.
func AssignTxHashes(blobs []*AgnosticBlobSidecar, blockTxs []bellatrix.Transaction) error {

	// The block's commitments, flattened in transaction order: one entry per
	// blob, holding the transaction that carried it and the hash it committed.
	type commitment struct {
		txHash   common.Hash
		blobHash string
	}

	committed := make([]commitment, 0, len(blobs))
	carriers := 0
	for idx, raw := range blockTxs {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(raw); err != nil {
			return fmt.Errorf("transaction %d could not be decoded: %w", idx, err)
		}
		if tx.Type() != types.BlobTxType {
			continue
		}
		carriers++
		for _, blobHash := range tx.BlobHashes() {
			committed = append(committed, commitment{tx.Hash(), blobHash.String()})
		}
	}

	if len(committed) != len(blobs) {
		return fmt.Errorf("%d blob sidecars but %d blobs committed by %d transactions",
			len(blobs), len(committed), carriers)
	}

	seen := make([]bool, len(committed))
	for _, blob := range blobs {
		// Compared before narrowing, not after. Index is a uint64, and
		// converting one at or above 2^63 to int wraps it negative, which slips
		// past a "too large" check and then panics indexing seen.
		if uint64(blob.Index) >= uint64(len(committed)) {
			return fmt.Errorf("blob index %d is past the %d blobs this block committed",
				blob.Index, len(committed))
		}
		index := int(blob.Index)
		if seen[index] {
			return fmt.Errorf("two sidecars claim blob index %d", blob.Index)
		}
		seen[index] = true

		if committed[index].blobHash != blob.BlobHash {
			// Both hashes belong in the message. Every other check here reports
			// the numbers it compared, and this is the one most likely to need
			// reading against the chain: when several blobs in a block share a
			// versioned hash, "does not match" alone gives nobody enough to
			// tell which sidecar arrived where it should not have.
			return fmt.Errorf("blob %d carries hash %s but this block committed %s at that position",
				blob.Index, blob.BlobHash, committed[index].blobHash)
		}
	}

	for _, blob := range blobs {
		blob.TxHash = committed[blob.Index].txHash
	}

	return nil
}

// DataColumnSidecarEventWrapper carries a data-column arrival plus the local time
// we observed it. Since the Fulu fork (PeerDAS) blobs are no longer gossiped whole:
// they are erasure-coded into columns and each node custodies a subset, so
// `data_column_sidecar` replaced the now-dead `blob_sidecar` event as the p2p
// propagation signal.
type DataColumnSidecarEventWrapper struct {
	Timestamp              time.Time
	DataColumnSidecarEvent api.DataColumnSidecarEvent
}

func KZGCommitmentToVersionedHash(input deneb.KZGCommitment) string {
	h := sha256.New()
	h.Write(input[:])
	sha256_hash := hex.EncodeToString(h.Sum(nil))

	return fmt.Sprintf("%s%s", versionedHashVersionKZG, sha256_hash[2:])
}
