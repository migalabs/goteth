CREATE TABLE IF NOT EXISTS t_head_events(
	f_slot INT,
    f_block TEXT,
    f_state TEXT,
    f_epoch_transition BOOLEAN,
    f_current_duty_dependent_root TEXT,
    f_previous_duty_dependent_root TEXT,
	f_arrival_timestamp BIGINT,
	CONSTRAINT PK_Block PRIMARY KEY (f_block));