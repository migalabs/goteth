package model

// Postgres intregration variables
var (
	CreateEpochMetricsTable = `
	CREATE TABLE IF NOT EXISTS t_epoch_metrics_summary(
		f_epoch INT,
		f_slot INT,
		f_num_att INT,
		f_num_att_vals INT,
		f_num_vals INT,
		f_total_balance BIGINT,
		f_total_effective_balance BIGINT,
		f_missing_source BIGINT, 
		f_missing_target BIGINT,
		f_missing_head BIGINT,
		f_missed_blocks BIGINT,
		CONSTRAINT PK_Epoch PRIMARY KEY (f_slot));`

	InsertNewEpochLineTable = `
	INSERT INTO t_epoch_metrics_summary (
		f_epoch, 
		f_slot, 
		f_num_att, 
		f_num_att_vals, 
		f_num_vals, 
		f_total_balance, 
		f_total_effective_balance, 
		f_missing_source, 
		f_missing_target, 
		f_missing_head, 
		f_missed_blocks)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);
	`

	UpdateRow = `
	UPDATE t_epoch_metrics_summary 
	SET 
		f_num_att = $2, 
		f_num_att_vals = $3, 
		f_num_vals = $4,
		f_total_balance = $5, 
		f_total_effective_balance = $6, 
		f_missing_source = $7, 
		f_missing_target = $8, 
		f_missing_head = $9
	WHERE f_slot=$1;
	`

	EPOCH_QUERIES = [...]string{InsertNewEpochLineTable, UpdateRow}
)

type EpochMetrics struct {
	Epoch                 uint64
	Slot                  uint64
	PrevNumAttestations   uint64
	PrevNumAttValidators  uint64
	PrevNumValidators     uint64
	TotalBalance          uint64
	TotalEffectiveBalance uint64

	MissingSource uint64
	MissingTarget uint64
	MissingHead   uint64

	MissedBlocks uint64
}

func NewEpochMetrics(iEpoch uint64,
	iSlot uint64,
	iNumAtt uint64,
	iNumAttVals uint64,
	iNumVals uint64,
	iTotBal uint64,
	iTotEfBal uint64,
	iMissingSource uint64,
	iMissingTarget uint64,
	iMissingHead uint64,
	iMissedBlocks uint64) EpochMetrics {
	return EpochMetrics{
		Epoch:                 iEpoch,
		Slot:                  iSlot,
		PrevNumAttestations:   iNumAtt,
		PrevNumAttValidators:  iNumVals,
		PrevNumValidators:     iNumVals,
		TotalBalance:          iTotBal,
		TotalEffectiveBalance: iTotEfBal,
		MissingSource:         iMissingSource,
		MissingTarget:         iMissingTarget,
		MissingHead:           iMissingHead,
		MissedBlocks:          iMissedBlocks,
	}
}

func NewEmptyEpochMetrics() EpochMetrics {
	return EpochMetrics{}
}
