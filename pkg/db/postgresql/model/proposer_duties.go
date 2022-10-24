package model

// Postgres intregration variables
var (
	CreateProposerDutiesTable = `
	CREATE TABLE IF NOT EXISTS t_proposer_duties(
		f_val_idx INT,
		f_proposer_slot INT,
		CONSTRAINT PK_Val_Slot PRIMARY KEY (f_val_idx, f_proposer_slot));`

	InsertProposerDuty = `
	INSERT INTO t_proposer_duties (
		f_val_idx, 
		f_proposer_slot)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING;
	`
	// if there is a confilct the line already exists
)

type ProposerDuties struct {
	ValIdx       uint64
	ProposerSlot uint64
}

func NewProposerDuties(
	iValIdx uint64,
	iProposerSlot uint64) ProposerDuties {

	return ProposerDuties{
		ValIdx:       iValIdx,
		ProposerSlot: iProposerSlot,
	}
}

func NewEmptyProposerDuties() ProposerDuties {
	return ProposerDuties{}
}
