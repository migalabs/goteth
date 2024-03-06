package clientapi

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
	local_spec "github.com/migalabs/goteth/pkg/spec"
)

func (s *APIClient) RequestBlobSidecars(slot phase0.Slot, txs []spec.AgnosticTransaction) ([]*local_spec.AgnosticBlobSidecar, error) {

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
