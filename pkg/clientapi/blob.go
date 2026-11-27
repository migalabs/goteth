package clientapi

import (
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	local_spec "github.com/migalabs/goteth/pkg/spec"
	"github.com/migalabs/goteth/pkg/utils"
)

func (s *APIClient) RequestBlobSidecars(slot phase0.Slot) ([]*local_spec.AgnosticBlobSidecar, error) {

	agnosticBlobs := make([]*local_spec.AgnosticBlobSidecar, 0)

	blobsResp, err := s.Api.BlobSidecars(s.ctx, &api.BlobSidecarsOpts{
		Block: fmt.Sprintf("%d", slot),
	})

	if err != nil {
		if isNotFound(err) {
			return agnosticBlobs, nil
		}
		return nil, fmt.Errorf("could not retrieve blob sidecars for slot %d: %s", slot, err)
	}

	blobs := blobsResp.Data

	for _, item := range blobs {
		agnosticBlob, err := local_spec.NewAgnosticBlobFromAPI(slot, *item)

		if err != nil {
			return nil, fmt.Errorf("could not retrieve blob sidecars for slot %d: %s", slot, err)
		}
		agnosticBlobs = append(agnosticBlobs, agnosticBlob)

	}
	return agnosticBlobs, nil
}

// requestKZGCommitmentFromSignedBlock fetches a block from /eth/v2/beacon/blocks/
// to match the KZG Commitments to the blobs given by the new blobs endpoint.
func (s *APIClient) requestKZGCommitmentFromSignedBlock(slot phase0.Slot, blockID string) ([]deneb.KZGCommitment, error) {
	resp, err := s.Api.SignedBeaconBlock(s.ctx, &api.SignedBeaconBlockOpts{
		Block: blockID,
	})

	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("could not retrieve KZGCommitments for slot %d: %s", slot, err)
	}

	return resp.Data.BlobKZGCommitments()
}

// RequestFuluBlobs uses the new endpoint /eth/v1/beacon/blobs/{block_id}
//
// The blobs and the block's KZG commitments come from two requests. Both ask for the block root
// (the slot only for a block without root), so a reorg between them cannot pair blobs with the
// commitments of another block. If the block request gets a 404 (the node can announce a block
// before serving it) the counts differ: both requests are retried, and if they still disagree the
// slot's blobs are reported as an error instead of indexing past the commitments.
func (s *APIClient) RequestFuluBlobs(slot phase0.Slot, root phase0.Root) ([]*local_spec.AgnosticBlobSidecar, error) {
	blobs := make([]*local_spec.AgnosticBlobSidecar, 0)
	blockID := fmt.Sprintf("%d", slot)
	if root != (phase0.Root{}) {
		blockID = fmt.Sprintf("%#x", root)
	}

	for attempt := 1; ; attempt++ {
		resp, err := s.Api.Blobs(s.ctx, &api.BlobsOpts{
			Block: blockID,
		})
		if err != nil {
			if isNotFound(err) {
				return blobs, nil
			}
			return nil, fmt.Errorf("could not retrieve blobs for slot %d: %s", slot, err)
		}

		kzgCommitments, err := s.requestKZGCommitmentFromSignedBlock(slot, blockID)
		if err != nil {
			return nil, err
		}

		if len(kzgCommitments) == len(resp.Data) {
			return buildFuluBlobs(slot, resp.Data, kzgCommitments), nil
		}
		if attempt >= s.maxRetries {
			return nil, fmt.Errorf("slot %d: %d blobs but %d KZG commitments in block %s after %d attempts",
				slot, len(resp.Data), len(kzgCommitments), blockID, attempt)
		}
		log.Debugf("slot %d: %d blobs but %d KZG commitments in block %s, retrying (attempt %d)",
			slot, len(resp.Data), len(kzgCommitments), blockID, attempt)
		time.Sleep(500 * time.Millisecond)
	}
}

// buildFuluBlobs pairs each blob with its KZG commitment; both slices have the same length.
func buildFuluBlobs(slot phase0.Slot, data []*deneb.Blob, kzgCommitments []deneb.KZGCommitment) []*local_spec.AgnosticBlobSidecar {
	blobs := make([]*local_spec.AgnosticBlobSidecar, 0, len(data))
	for i, blob := range data {
		var b local_spec.AgnosticBlobSidecar

		b.Blob = *blob
		b.Index = deneb.BlobIndex(i)
		b.Slot = slot
		b.BlobEnding0s = utils.CountConsecutiveEnding0(b.Blob[:])
		b.KZGCommitment = kzgCommitments[i]
		b.BlobHash = local_spec.KZGCommitmentToVersionedHash(b.KZGCommitment)
		blobs = append(blobs, &b)
	}
	return blobs
}
