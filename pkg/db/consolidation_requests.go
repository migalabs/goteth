package db

import (
	"github.com/ClickHouse/ch-go/proto"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	consolidationRequestsTable      = "t_consolidation_requests"
	insertConsolidationRequestQuery = `
	INSERT INTO %s (
		f_slot,
		f_index,
		f_source_address,
		f_source_pubkey,
		f_target_pubkey,
		f_result
		)
		VALUES`
)

func consolidationRequestsInput(consolidationRequestss []spec.ConsolidationRequest) proto.Input {
	// one object per column
	var (
		f_slot           proto.ColUInt64
		f_index          proto.ColUInt64
		f_source_address proto.ColStr
		f_source_pubkey  proto.ColStr
		f_target_pubkey  proto.ColStr
		f_result         proto.ColUInt8
	)

	for _, consolidationRequest := range consolidationRequestss {

		f_slot.Append(uint64(consolidationRequest.Slot))
		f_index.Append(consolidationRequest.Index)
		f_source_address.Append(consolidationRequest.SourceAddress.String())
		f_source_pubkey.Append(consolidationRequest.SourcePubkey.String())
		f_target_pubkey.Append(consolidationRequest.TargetPubkey.String())
		f_result.Append(uint8(consolidationRequest.Result))
	}

	return proto.Input{
		{Name: "f_slot", Data: f_slot},
		{Name: "f_index", Data: f_index},
		{Name: "f_source_address", Data: f_source_address},
		{Name: "f_source_pubkey", Data: f_source_pubkey},
		{Name: "f_target_pubkey", Data: f_target_pubkey},
		{Name: "f_result", Data: f_result},
	}
}

func (p *DBService) PersistConsolidationRequests(data []spec.ConsolidationRequest) error {
	persistObj := PersistableObject[spec.ConsolidationRequest]{
		input: consolidationRequestsInput,
		table: consolidationRequestsTable,
		query: insertConsolidationRequestQuery,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting consolidationRequests: %s", err.Error())
	}
	return err
}
