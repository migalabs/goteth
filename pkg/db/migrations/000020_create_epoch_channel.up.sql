CREATE OR REPLACE FUNCTION notify_epoch_insert() RETURNS TRIGGER AS $$ 
	DECLARE 
	row RECORD; 
	output TEXT;
	last_epoch integer;
	last_validator integer;
    BEGIN 
    row = NEW; 

	-- Check if the epoch is greater than the previous one
	SELECT f_epoch INTO last_epoch
	FROM t_epoch_metrics_summary
	order by f_epoch desc
	limit 1;

	IF row.f_epoch > last_epoch THEN
	    -- Forming the Output as notification. You can choose you own notification. 
		output = 'OPERATION = ' || TG_OP || ' and Epoch = ' || row.f_epoch; 
		-- Calling the pg_notify for my_table_update event with output as payload 
		PERFORM pg_notify('new_epoch_finalized',output); 
	END IF;

	SELECT f_num_vals INTO last_validator
	FROM t_epoch_metrics_summary
	order by f_epoch desc
	limit 1;

	IF row.f_num_vals > last_validator THEN
		output = 'Validator Num = ' || row.f_num_vals; 
		PERFORM pg_notify('new_validator', output); 
	END IF;


    -- Returning null because it is an after trigger. 
    RETURN NEW; 
    END; 
$$ LANGUAGE plpgsql; 

CREATE TRIGGER trigger_notify_new_epoch
  BEFORE INSERT 
  ON t_epoch_metrics_summary 
  FOR EACH ROW 
  EXECUTE PROCEDURE notify_epoch_insert();