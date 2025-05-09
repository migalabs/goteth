package clientapi

import (
	"context"
	"testing"

	"github.com/migalabs/goteth/pkg/spec"
	"github.com/stretchr/testify/assert"
)

// https://github.com/ethereum/consensus-specs/blob/dev/specs/deneb/beacon-chain.md#kzg_commitment_to_versioned_hash
func TestBlobHash(t *testing.T) {
	maxRequestRetries := 3
	cli, err := NewAPIClient(context.Background(), "http://localhost:5052", "", maxRequestRetries)
	if err != nil {
		return
	}
	blobs, err := cli.RequestBlobSidecars(1167743)

	for _, blob := range blobs {
		if blob.Index == 1 {
			realHash := "0x011ee86e83951989dcf96af5c3aeae51ecd15575f8c0003cb5a275945322e98d"
			versionedHash := spec.KZGCommitmentToVersionedHash(blob.KZGCommitment)

			assert.Equal(t, realHash, versionedHash)

		}
	}
}
