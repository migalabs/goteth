package clientapi

import (
	"fmt"

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
		if response404(err.Error()) {
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

func (s *APIClient) requestKZGCommitmentFromSignedBlock(slot phase0.Slot) ([]deneb.KZGCommitment, error) {
	resp, err := s.Api.SignedBeaconBlock(s.ctx, &api.SignedBeaconBlockOpts{
		Block: fmt.Sprintf("%d", slot),
	})

	if err != nil {
		if response404(err.Error()) {
			return nil, nil
		}
		return nil, fmt.Errorf("could not retrieve KZGCommitments for slot %d: %s", slot, err)
	}

	return resp.Data.BlobKZGCommitments()
}

// RequestFuluBlobs uses the new endpoint /eth/v1/beacon/blobs/{block_id}
func (s *APIClient) RequestFuluBlobs(slot phase0.Slot) ([]*local_spec.AgnosticBlobSidecar, error) {
	blobs := make([]*local_spec.AgnosticBlobSidecar, 0)

	resp, err := s.Api.Blobs(s.ctx, &api.BlobsOpts{
		Block: fmt.Sprintf("%d", slot),
	})

	if err != nil {
		if response404(err.Error()) {
			return blobs, nil
		}
		return nil, fmt.Errorf("could not retrieve blobs for slot %d: %s", slot, err)
	}

	var kzgCommitments []deneb.KZGCommitment
	kzgCommitments, err = s.requestKZGCommitmentFromSignedBlock(slot)

	if err != nil {
		return nil, err
	}

	for i, blob := range resp.Data {
		var b local_spec.AgnosticBlobSidecar

		b.Blob = *blob
		b.Index = deneb.BlobIndex(i)
		b.Slot = slot
		b.BlobEnding0s = utils.CountConsecutiveEnding0(b.Blob[:])
		b.KZGCommitment = kzgCommitments[i]
		b.BlobHash = local_spec.KZGCommitmentToVersionedHash(b.KZGCommitment)
		blobs = append(blobs, &b)
	}

	return blobs, nil
}
