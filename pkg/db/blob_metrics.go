package db

import (
	"github.com/ClickHouse/ch-go/proto"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	blobsTable              = "t_blob_sidecars"
	insertBlobSidecarsQuery = `
	INSERT INTO %s (
		f_blob_hash,
		f_tx_hash,
		f_slot,
		f_index,
		f_kzg_commitment,
		f_kzg_proof)
		VALUES`

//	deleteBlockQuery = `
//		DELETE FROM %s
//		WHERE f_slot = $1;
//
// `
)

func blobSidecarsInput(blobSidecars []spec.AgnosticBlobSidecar) proto.Input {
	// one object per column
	var (
		f_blob_hash      proto.ColStr
		f_tx_hash        proto.ColStr
		f_slot           proto.ColUInt64
		f_index          proto.ColUInt8
		f_kzg_commitment proto.ColStr
		f_kzg_proof      proto.ColStr
	)

	for _, blobSidecar := range blobSidecars {

		f_blob_hash.Append(blobSidecar.BlobHash)
		f_tx_hash.Append(blobSidecar.TxHash.String())
		f_slot.Append(uint64(blobSidecar.Slot))
		f_index.Append(uint8(blobSidecar.Index))
		f_kzg_commitment.Append(blobSidecar.KZGCommitment.String())
		f_kzg_proof.Append(blobSidecar.KZGProof.String())

	}

	return proto.Input{

		{Name: "f_blob_hash", Data: f_blob_hash},
		{Name: "f_tx_hash", Data: f_tx_hash},
		{Name: "f_slot", Data: f_slot},
		{Name: "f_index", Data: f_index},
		{Name: "f_kzg_commitment", Data: f_kzg_commitment},
		{Name: "f_kzg_proof", Data: f_kzg_proof},
	}
}

func (p *DBService) PersistBlobSidecars(data []*spec.AgnosticBlobSidecar) error {
	persistObj := PersistableObject[spec.AgnosticBlobSidecar]{
		input: blobSidecarsInput,
		table: blobsTable,
		query: insertBlobSidecarsQuery,
	}

	for _, item := range data {
		persistObj.Append(*item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting blobs: %s", err.Error())
	}
	return err
}
