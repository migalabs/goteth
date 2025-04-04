package db

import (
	"github.com/ClickHouse/ch-go/proto"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	consolidationsProcessedTable       = "t_consolidations_processed"
	insertConsolidationsProcessedQuery = `
	INSERT INTO %s (
		f_epoch,
		f_index,
		f_source_index,
		f_target_index,
		f_consolidated_amount,
		f_valid
		)
		VALUES`
)

func consolidationsProcessedInput(consolidationsProcessed []spec.ConsolidationProcessed) proto.Input {
	// one object per column
	var (
		f_epoch               proto.ColUInt64
		f_index               proto.ColUInt64
		f_source_index        proto.ColUInt64
		f_target_index        proto.ColUInt64
		f_consolidated_amount proto.ColUInt64
		f_valid               proto.ColBool
	)

	for _, consolidationProcessed := range consolidationsProcessed {
		f_epoch.Append(uint64(consolidationProcessed.Epoch))
		f_index.Append(uint64(consolidationProcessed.Index))
		f_source_index.Append(uint64(consolidationProcessed.SourceIndex))
		f_target_index.Append(uint64(consolidationProcessed.TargetIndex))
		f_consolidated_amount.Append(uint64(consolidationProcessed.ConsolidatedAmount))
		f_valid.Append(consolidationProcessed.Valid)
	}

	return proto.Input{
		{Name: "f_epoch", Data: f_epoch},
		{Name: "f_index", Data: f_index},
		{Name: "f_source_index", Data: f_source_index},
		{Name: "f_target_index", Data: f_target_index},
		{Name: "f_consolidated_amount", Data: f_consolidated_amount},
		{Name: "f_valid", Data: f_valid},
	}
}

func (p *DBService) PersistConsolidationsProcessed(data []spec.ConsolidationProcessed) error {
	persistObj := PersistableObject[spec.ConsolidationProcessed]{
		input: consolidationsProcessedInput,
		table: consolidationsProcessedTable,
		query: insertConsolidationsProcessedQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting consolidationsProcessed: %s", err.Error())
	}
	return err
}
