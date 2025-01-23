package db

import (
	"github.com/ClickHouse/ch-go/proto"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	blobEventsTable               = "t_blob_sidecars_events"
	insertBlobSideCarsEventsQuery = `
	INSERT INTO %s (
		f_arrival_timestamp_ms,
		f_blob_hash,
		f_slot,
		f_block_root,
		f_index,
		f_kzg_commitment)
		VALUES`
)

func blobSidecarsEventInput(blobSidecarsEvents []spec.BlobSideCarEventWraper) proto.Input {
	// one object per column
	var (
		f_arrival_timestamp_ms proto.ColUInt64
		f_blob_hash            proto.ColStr
		f_slot                 proto.ColUInt64
		f_block_root           proto.ColStr
		f_index                proto.ColUInt8
		f_kzg_commitment       proto.ColStr
	)

	for _, blobSidecar := range blobSidecarsEvents {

		f_arrival_timestamp_ms.Append(uint64(blobSidecar.Timestamp.UnixMilli()))
		f_blob_hash.Append(blobSidecar.BlobSidecarEvent.VersionedHash.String())
		f_slot.Append(uint64(blobSidecar.BlobSidecarEvent.Slot))
		f_block_root.Append(blobSidecar.BlobSidecarEvent.BlockRoot.String())
		f_index.Append(uint8(blobSidecar.BlobSidecarEvent.Index))
		f_kzg_commitment.Append(blobSidecar.BlobSidecarEvent.KZGCommitment.String())
	}

	return proto.Input{

		{Name: "f_arrival_timestamp_ms", Data: f_arrival_timestamp_ms},
		{Name: "f_blob_hash", Data: f_blob_hash},
		{Name: "f_slot", Data: f_slot},
		{Name: "f_block_root", Data: f_block_root},
		{Name: "f_index", Data: f_index},
		{Name: "f_kzg_commitment", Data: f_kzg_commitment},
	}
}

func (p *DBService) PersistBlobSidecarsEvents(data []spec.BlobSideCarEventWraper) error {
	persistObj := PersistableObject[spec.BlobSideCarEventWraper]{
		input: blobSidecarsEventInput,
		table: blobEventsTable,
		query: insertBlobSideCarsEventsQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting blob events: %s", err.Error())
	}
	return err
}
