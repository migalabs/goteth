package model

// Postgres intregration variables
var (
	CreateProposerDutiesTable = `
	CREATE TABLE IF NOT EXISTS t_proposer_duties(
		f_val_idx INT,
		f_proposer_slot INT,
		f_proposed BOOL,
		CONSTRAINT PK_Val_Slot PRIMARY KEY (f_val_idx, f_proposer_slot));`

	InsertProposerDuty = `
	INSERT INTO t_proposer_duties (
		f_val_idx, 
		f_proposer_slot,
		f_proposed)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING;
	`
	// if there is a confilct the line already exists
)

type ProposerDuties struct {
	ValIdx       uint64
	ProposerSlot uint64
	Proposed     bool
}

func NewProposerDuties(
	iValIdx uint64,
	iProposerSlot uint64,
	iProposed bool) ProposerDuties {

	return ProposerDuties{
		ValIdx:       iValIdx,
		ProposerSlot: iProposerSlot,
		Proposed:     iProposed,
	}
}

func NewEmptyProposerDuties() ProposerDuties {
	return ProposerDuties{}
}
