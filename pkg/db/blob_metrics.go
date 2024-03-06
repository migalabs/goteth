package db

import (
	"github.com/ClickHouse/ch-go/proto"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	blobsTable              = "t_blob_sidecars"
	insertBlobSidecarsQuery = `
	INSERT INTO %s (
		f_arrival_timestamp_ms,
		f_blob_hash,
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
		f_arrival_timestamp_ms proto.ColUInt64
		f_blob_hash            proto.ColStr
		f_slot                 proto.ColUInt64
		f_index                proto.ColUInt8
		f_kzg_commitment       proto.ColStr
		f_kzg_proof            proto.ColStr
	)

	for _, blobSidecar := range blobSidecars {

		f_arrival_timestamp_ms.Append(uint64(blobSidecar.ArrivalTimestamp.UnixMilli()))
		f_blob_hash.Append(blobSidecar.BlobHash.String())
		f_slot.Append(uint64(blobSidecar.Slot))
		f_index.Append(uint8(blobSidecar.Index))
		f_kzg_commitment.Append(blobSidecar.KZGCommitment.String())
		f_kzg_proof.Append(blobSidecar.KZGProof.String())

	}

	return proto.Input{

		{Name: "f_arrival_timestamp_ms", Data: f_arrival_timestamp_ms},
		{Name: "f_blob_hash", Data: f_blob_hash},
		{Name: "f_slot", Data: f_slot},
		{Name: "f_index", Data: f_index},
		{Name: "f_kzg_commitment", Data: f_kzg_commitment},
		{Name: "f_kzg_proof", Data: f_kzg_proof},
	}
}

func (p *DBService) PersistBlobSidecars(data []spec.AgnosticBlobSidecar) error {
	persistObj := PersistableObject[spec.AgnosticBlobSidecar]{
		input: blobSidecarsInput,
		table: blobsTable,
		query: insertBlobSidecarsQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting blobs: %s", err.Error())
	}
	return err
}

// func (s *DBService) DeleteBlockMetrics(slot phase0.Slot) error {

// 	err := s.Delete(DeletableObject{
// 		query: deleteBlockQuery,
// 		table: blocksTable,
// 		args:  []any{slot},
// 	})
// 	if err != nil {
// 		return err
// 	}

// 	err = s.Delete(DeletableObject{
// 		query: deleteTransactionsQuery,
// 		table: transactionsTable,
// 		args:  []any{slot},
// 	})
// 	if err != nil {
// 		return err
// 	}
// 	err = s.Delete(DeletableObject{
// 		query: deleteWithdrawalsQuery,
// 		table: withdrawalsTable,
// 		args:  []any{slot},
// 	})
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (p *DBService) RetrieveLastSlot() (phase0.Slot, error) {

// 	var dest []struct {
// 		F_slot uint64 `ch:"f_slot"`
// 	}

// 	err := p.highSelect(
// 		fmt.Sprintf(selectLastSlotQuery, blocksTable),
// 		&dest)

// 	if len(dest) > 0 {
// 		return phase0.Slot(dest[0].F_slot), err
// 	}
// 	return 0, err

// }
