DROP TRIGGER IF EXISTS trigger_notify_new_epoch ON t_epoch_metrics_summary;
DROP function IF EXISTS notify_epoch_insert;

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
	
	INSERT INTO t_pool_summary
		SELECT 
			t_eth2_pubkeys.f_pool_name, f_epoch,
			SUM(CASE WHEN (f_reward <= f_max_reward) THEN f_reward ELSE 0 END) as aggregated_rewards,
			SUM(CASE WHEN (f_reward <= f_max_reward) THEN f_max_reward ELSE 0 END) as aggregated_max_rewards,
			COUNT(CASE WHEN f_in_sync_committee = TRUE THEN 1 ELSE null END) as count_sync_committee,
			COUNT(CASE WHEN f_missing_source = TRUE THEN 1 ELSE null END) as count_missing_source,
			COUNT(CASE WHEN f_missing_target = TRUE THEN 1 ELSE null END) as count_missing_target,
			COUNT(CASE WHEN f_missing_head = TRUE THEN 1 ELSE null END) as count_missing_head,
			COUNT(*) as count_expected_attestations,
			SUM(CASE WHEN t_proposer_duties.f_proposed = TRUE THEN 1 ELSE 0 END) as proposed_blocks_performance,
			SUM(CASE WHEN t_proposer_duties.f_proposed = FALSE THEN 1 ELSE 0 END) as missed_blocks_performance,
			count(distinct(t_validator_rewards_summary.f_val_idx)) as number_active_vals
		FROM t_validator_rewards_summary
		LEFT JOIN t_proposer_duties 
			ON t_validator_rewards_summary.f_val_idx = t_proposer_duties.f_val_idx 
			AND t_validator_rewards_summary.f_epoch = t_proposer_duties.f_proposer_slot/32
		INNER JOIN t_eth2_pubkeys 
			ON t_validator_rewards_summary.f_val_idx = t_eth2_pubkeys.f_val_idx
		WHERE f_epoch = row.f_epoch AND f_status IN (1, 3)
		GROUP BY t_eth2_pubkeys.f_pool_name, f_epoch
	ON CONFLICT DO NOTHING;
		
		
    -- Returning null because it is an after trigger. 
    RETURN NEW; 
    END; 
$$ LANGUAGE plpgsql; 


CREATE TRIGGER trigger_notify_new_epoch
  BEFORE INSERT 
  ON t_epoch_metrics_summary 
  FOR EACH ROW 
  EXECUTE PROCEDURE notify_epoch_insert();