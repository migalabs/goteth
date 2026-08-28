package spec

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	api "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
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
// Nothing is assigned until the whole block has been checked: the sidecars must
// account for exactly the blobs the transactions committed to, each index must
// appear once, and every hash must agree with the one committed at that
// position. Any disagreement means these sidecars are not the ones this block's
// transactions committed to, or that a transaction went missing because its
// receipt never arrived, and the positions are then meaningless. In that case
// the mapping is left untouched and an error returned, rather than shifting
// every blob onto the wrong transaction.
func AssignTxHashes(blobs []*AgnosticBlobSidecar, txs []AgnosticTransaction) error {

	carriers := make([]AgnosticTransaction, 0, len(blobs))

	for _, tx := range txs {
		if len(tx.BlobHashes) == 0 {
			continue // this tx does not reference any blobs
		}
		carriers = append(carriers, tx)
	}

	sort.Slice(carriers, func(i, j int) bool { return carriers[i].TxIdx < carriers[j].TxIdx })

	// The block's commitments, flattened in transaction order: one entry per
	// blob, holding the transaction that carried it and the hash it committed.
	type commitment struct {
		txHash   common.Hash
		blobHash string
	}
	committed := make([]commitment, 0, len(blobs))
	for _, tx := range carriers {
		for _, blobHash := range tx.BlobHashes {
			committed = append(committed, commitment{common.Hash(tx.Hash), blobHash.String()})
		}
	}

	if len(committed) != len(blobs) {
		return fmt.Errorf("%d blob sidecars but %d blobs committed by %d transactions",
			len(blobs), len(committed), len(carriers))
	}

	seen := make([]bool, len(committed))
	for _, blob := range blobs {
		index := int(blob.Index)
		if index >= len(committed) {
			return fmt.Errorf("blob index %d is past the %d blobs this block committed",
				blob.Index, len(committed))
		}
		if seen[index] {
			return fmt.Errorf("two sidecars claim blob index %d", blob.Index)
		}
		seen[index] = true

		if committed[index].blobHash != blob.BlobHash {
			return fmt.Errorf("blob %d does not match the hash committed at that position",
				blob.Index)
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
