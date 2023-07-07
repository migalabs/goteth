CREATE TABLE IF NOT EXISTS t_finalized_checkpoint(
	f_id INT PRIMARY KEY,
	f_block_root TEXT,
	f_state_root TEXT,
	f_epoch INT);