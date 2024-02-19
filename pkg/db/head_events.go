package db

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

import (
	"github.com/ClickHouse/ch-go/proto"
	api "github.com/attestantio/go-eth2-client/api/v1"
)

// Postgres intregration variables
var (
	headEventsTable  = "t_head_events"
	insertHeadEvents = `
	INSERT INTO %s (
		f_slot,
		f_block,
		f_state,
		f_epoch_transition,
		f_current_duty_dependent_root,
		f_previous_duty_dependent_root,
		f_arrival_timestamp)
		VALUES`
)

func headEventsInput(events []HeadEvent) proto.Input {
	// one object per column
	var (
		f_slot                         proto.ColUInt64
		f_block                        proto.ColStr
		f_state                        proto.ColStr
		f_epoch_transition             proto.ColBool
		f_current_duty_dependent_root  proto.ColStr
		f_previous_duty_dependent_root proto.ColStr
		f_arrival_timestamp            proto.ColUInt64
	)

	for _, event := range events {

		f_slot.Append(uint64(event.HeadEvent.Slot))
		f_block.Append(event.HeadEvent.Block.String())
		f_state.Append(event.HeadEvent.State.String())
		f_epoch_transition.Append(event.HeadEvent.EpochTransition)
		f_current_duty_dependent_root.Append(event.HeadEvent.CurrentDutyDependentRoot.String())
		f_previous_duty_dependent_root.Append(event.HeadEvent.PreviousDutyDependentRoot.String())
		f_arrival_timestamp.Append(uint64(event.ArrivalTimestamp))
	}

	return proto.Input{

		{Name: "f_slot", Data: f_slot},
		{Name: "f_block", Data: f_block},
		{Name: "f_state", Data: f_state},
		{Name: "f_epoch_transition", Data: f_epoch_transition},
		{Name: "f_current_duty_dependent_root", Data: f_current_duty_dependent_root},
		{Name: "f_previous_duty_dependent_root", Data: f_previous_duty_dependent_root},
		{Name: "f_arrival_timestamp", Data: f_arrival_timestamp},
	}
}

func (p *DBService) PersistHeadEvents(data []HeadEvent) error {
	persistObj := PersistableObject[HeadEvent]{
		input: headEventsInput,
		table: headEventsTable,
		query: insertHeadEvents,
	}

	for _, item := range data {
		persistObj.Append(item)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting head events: %s", err.Error())
	}
	return err
}

type HeadEvent struct {
	HeadEvent        api.HeadEvent
	ArrivalTimestamp int64
}
