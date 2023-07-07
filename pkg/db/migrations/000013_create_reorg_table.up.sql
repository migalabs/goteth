CREATE TABLE IF NOT EXISTS t_reorgs(
	f_slot INT PRIMARY KEY,
	f_depth INT,
	f_old_head_block_root TEXT,
	f_new_head_block_root TEXT,
	f_old_head_state_root TEXT,
	f_new_head_state_root TEXT);