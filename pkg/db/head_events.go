package db

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

import (
	api "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/migalabs/goteth/pkg/spec"
)

// Postgres intregration variables
var (
	UpsertHeadEvent = `
	INSERT INTO t_head_events (
		f_slot,
		f_block,
		f_state,
		f_epoch_transition,
		f_current_duty_dependent_root,
		f_previous_duty_dependent_root,
		f_arrival_timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT ON CONSTRAINT PK_Block
		DO 
			UPDATE SET
				f_slot = excluded.f_slot, 
				f_block = excluded.f_block,
				f_state = excluded.f_state,
				f_epoch_transition = excluded.f_epoch_transition,
				f_current_duty_dependent_root = excluded.f_current_duty_dependent_root,
				f_previous_duty_dependent_root = excluded.f_previous_duty_dependent_root,
				f_arrival_timestamp = excluded.f_arrival_timestamp;
	`
)

func insertHeadEvent(inputHeadEvent api.HeadEvent, arrivalTimestamp int64) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)
	resultArgs = append(resultArgs, inputHeadEvent.Slot)
	resultArgs = append(resultArgs, inputHeadEvent.Block)
	resultArgs = append(resultArgs, inputHeadEvent.State)
	resultArgs = append(resultArgs, inputHeadEvent.EpochTransition)
	resultArgs = append(resultArgs, inputHeadEvent.CurrentDutyDependentRoot)
	resultArgs = append(resultArgs, inputHeadEvent.PreviousDutyDependentRoot)
	resultArgs = append(resultArgs, arrivalTimestamp)
	return UpsertHeadEvent, resultArgs
}

func HeadEventOperation(inputHeadEvent HeadEventType) (string, []interface{}) {
	q, args := insertHeadEvent(inputHeadEvent.HeadEvent, inputHeadEvent.ArrivalTimestamp)
	return q, args
}

type HeadEventType struct {
	HeadEvent        api.HeadEvent
	ArrivalTimestamp int64
}

func (s HeadEventType) Type() spec.ModelType {
	return spec.HeadEventModel
}

func HeadEventTypeFromHeadEvent(input api.HeadEvent, arrivalTimestamp int64) HeadEventType {
	return HeadEventType{
		HeadEvent:        input,
		ArrivalTimestamp: arrivalTimestamp,
	}
}
