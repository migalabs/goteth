package spec

import (
	"time"

	api "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
)

const (
	maxBlobsPerBlock int = 6
)

type AgnosticBlobSidecar struct {
	ArrivalTimestamp            time.Time
	Slot                        phase0.Slot
	TxHash                      common.Hash
	BlobHash                    common.Hash
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
		KZGProof:                    blob.KZGProof,
		SignedBlockHeader:           blob.SignedBlockHeader,
		KZGCommitmentInclusionProof: blob.KZGCommitmentInclusionProof,
	}, nil
}

func (b *AgnosticBlobSidecar) AddEventData(blobSidecarEvent BlobSideCarEventWraper) {
	b.BlobHash = common.Hash(blobSidecarEvent.BlobSidecarEvent.VersionedHash)
	b.ArrivalTimestamp = blobSidecarEvent.Timestamp
}

func (b *AgnosticBlobSidecar) GetTxHash(txs []AgnosticTransaction) {

	for _, tx := range txs {
		if len(tx.BlobHashes) == 0 {
			continue // this tx does not reference any blobs
		}

		for _, txBlobHash := range tx.BlobHashes {
			if txBlobHash == b.BlobHash {
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

type BlobSidecarsInSlot struct {
	Slot         phase0.Slot
	BlobSidecars map[int]*AgnosticBlobSidecar
}

func NewBlobSidecarsInSlot(slot phase0.Slot) *BlobSidecarsInSlot {
	return &BlobSidecarsInSlot{
		Slot:         slot,
		BlobSidecars: make(map[int]*AgnosticBlobSidecar, maxBlobsPerBlock),
	}
}

func (b *BlobSidecarsInSlot) AddNewBlobSidecar(blobSidecar *AgnosticBlobSidecar) {
	b.BlobSidecars[int(blobSidecar.Index)] = blobSidecar
}
