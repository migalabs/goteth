
CREATE OR REPLACE FUNCTION notify_head_insert() RETURNS TRIGGER AS $$ 
    DECLARE 
    row RECORD; 
    output TEXT; 
	last_slot integer;
    BEGIN 
	row = NEW; 

	-- Check if the epoch is greater than the previous one
	SELECT f_slot INTO last_slot
	FROM t_block_metrics
	order by f_slot desc
	limit 1;

	IF row.f_slot > last_slot THEN
		-- Forming the Output as notification. You can choose you own notification. 
		output = 'OPERATION = ' || TG_OP || ' and Slot = ' || row.f_slot; 
		-- Calling the pg_notify for my_table_update event with output as payload 
		PERFORM pg_notify('new_head', output); 
	END IF;
    -- Returning null because it is an after trigger. 
    RETURN NEW; 
    END; 
$$ LANGUAGE plpgsql; 
 

CREATE TRIGGER trigger_notify_new_head 
  BEFORE INSERT 
  ON t_block_metrics 
  FOR EACH ROW 
  EXECUTE PROCEDURE notify_head_insert(); 




