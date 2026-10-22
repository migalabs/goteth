package db

import (
	"github.com/ClickHouse/ch-go/proto"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	dataColumnEventsTable               = "t_data_column_sidecars_events"
	insertDataColumnSidecarsEventsQuery = `
	INSERT INTO %s (
		f_arrival_timestamp_ms,
		f_slot,
		f_block_root,
		f_column_index,
		f_kzg_commitments,
		f_num_commitments)
		VALUES`
)

func dataColumnSidecarsEventInput(events []spec.DataColumnSidecarEventWrapper) proto.Input {
	// one object per column
	var (
		f_arrival_timestamp_ms proto.ColUInt64
		f_slot                 proto.ColUInt64
		f_block_root           proto.ColStr
		f_column_index         proto.ColUInt16
		f_kzg_commitments      = proto.NewArray[string](new(proto.ColStr))
		f_num_commitments      proto.ColUInt16
	)

	for _, ev := range events {
		commitments := make([]string, 0, len(ev.DataColumnSidecarEvent.KZGCommitments))
		for _, c := range ev.DataColumnSidecarEvent.KZGCommitments {
			commitments = append(commitments, c.String())
		}

		f_arrival_timestamp_ms.Append(uint64(ev.Timestamp.UnixMilli()))
		f_slot.Append(uint64(ev.DataColumnSidecarEvent.Slot))
		f_block_root.Append(ev.DataColumnSidecarEvent.BlockRoot.String())
		f_column_index.Append(uint16(ev.DataColumnSidecarEvent.Index))
		f_kzg_commitments.Append(commitments)
		// Denormalised so "how many blobs did this block carry" does not have to
		// unnest the array on a table that takes ~128 rows per slot.
		f_num_commitments.Append(uint16(len(commitments)))
	}

	return proto.Input{
		{Name: "f_arrival_timestamp_ms", Data: f_arrival_timestamp_ms},
		{Name: "f_slot", Data: f_slot},
		{Name: "f_block_root", Data: f_block_root},
		{Name: "f_column_index", Data: f_column_index},
		{Name: "f_kzg_commitments", Data: f_kzg_commitments},
		{Name: "f_num_commitments", Data: f_num_commitments},
	}
}

func (p *DBService) PersistDataColumnSidecarsEvents(data []spec.DataColumnSidecarEventWrapper) error {
	persistObj := PersistableObject[spec.DataColumnSidecarEventWrapper]{
		input: dataColumnSidecarsEventInput,
		table: dataColumnEventsTable,
		query: insertDataColumnSidecarsEventsQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting data column events: %s", err.Error())
	}
	return err
}
