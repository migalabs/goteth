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
)

const (
	maxBlobsPerBlock int = 6
)

var (
	versionedHashVersionKZG = []byte("0x01")
)

type AgnosticBlobSidecar struct {
	Slot                        phase0.Slot
	TxHash                      common.Hash
	BlobHash                    string
	Blob                        deneb.Blob
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

type BlobSideCarEventWraper struct {
	Timestamp        time.Time
	BlobSidecarEvent api.BlobSidecarEvent
}

func KZGCommitmentToVersionedHash(input deneb.KZGCommitment) string {
	h := sha256.New()
	h.Write(input[:])
	sha256_hash := hex.EncodeToString(h.Sum(nil))

	return fmt.Sprintf("%s%s", versionedHashVersionKZG, sha256_hash[2:])
}
