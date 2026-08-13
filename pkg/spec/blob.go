package spec

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

func (b *AgnosticBlobSidecar) GetTxHash(txs []AgnosticTransaction) {

	for _, tx := range txs {
		if tx.BlobHashes == nil {
			continue // this tx does not reference any blobs
		}

		for _, txBlobHash := range tx.BlobHashes {
			if txBlobHash.String() == b.BlobHash {
				// we found it
				b.TxHash = common.Hash(tx.Hash)
			}
		}
	}
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
