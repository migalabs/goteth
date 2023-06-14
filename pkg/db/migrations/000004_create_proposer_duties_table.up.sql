CREATE TABLE IF NOT EXISTS t_proposer_duties(
	f_val_idx INT,
	f_proposer_slot INT,
	f_proposed BOOL,
	CONSTRAINT PK_Val_Slot PRIMARY KEY (f_val_idx, f_proposer_slot));