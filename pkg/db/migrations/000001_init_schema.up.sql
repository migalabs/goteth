
CREATE TABLE IF NOT EXISTS t_block_metrics(
	f_timestamp UInt64,
	f_epoch UInt64,
	f_slot UInt64,
	f_graffiti TEXT,
	f_proposer_index UInt64,
	f_proposed BOOL,
	f_attestations UInt64,
	f_deposits UInt64,
	f_proposer_slashings UInt64,
	f_attester_slashings UInt64,
	f_voluntary_exits UInt64,
	f_sync_bits UInt64,
	f_el_fee_recp TEXT,
	f_el_gas_limit UInt64,
	f_el_gas_used UInt64,
	f_el_base_fee_per_gas UInt64,
	f_el_block_hash TEXT,
	f_el_transactions UInt64,
	f_el_block_number UInt64,
	f_payload_size_bytes UInt64,
	f_ssz_size_bytes Float,
	f_snappy_size_bytes Float,
	f_compression_time_ms Float,
	f_decompression_time_ms Float)
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_slot);

CREATE TABLE IF NOT EXISTS t_orphans(
	f_timestamp UInt64,
	f_epoch UInt64,
	f_slot UInt64 PRIMARY KEY,
	f_graffiti TEXT,
	f_proposer_index UInt64,
	f_proposed BOOL,
	f_attestations UInt64,
	f_deposits UInt64,
	f_proposer_slashings UInt64,
	f_attester_slashings UInt64,
	f_voluntary_exits UInt64,
	f_sync_bits UInt64,
	f_el_fee_recp TEXT,
	f_el_gas_limit UInt64,
	f_el_gas_used UInt64,
	f_el_base_fee_per_gas UInt64,
	f_el_block_hash TEXT,
	f_el_transactions UInt64,
	f_el_block_number UInt64,
	f_payload_size_bytes UInt64,
	f_ssz_size_bytes Float,
	f_snappy_size_bytes Float,
	f_compression_time_ms Float,
	f_decompression_time_ms Float)
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_slot);

CREATE TABLE IF NOT EXISTS t_epoch_metrics_summary(
	f_timestamp UInt64,
	f_epoch UInt64,
	f_slot UInt64,
	f_num_att UInt64,
	f_num_att_vals UInt64,
	f_num_vals UInt64,
	f_total_balance_eth Float,
	f_att_effective_balance_eth Float,
	f_total_effective_balance_eth Float,
	f_missing_source UInt64, 
	f_missing_target UInt64,
	f_missing_head UInt64,
	f_num_slashed_vals UInt64 DEFAULT 0,
	f_num_active_vals UInt64 DEFAULT 0,
	f_num_exited_vals UInt64 DEFAULT 0,
	f_num_in_activation_vals UInt64 DEFAULT 0)
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_epoch);

CREATE TABLE IF NOT EXISTS t_pool_summary(
	f_pool_name TEXT,
	f_epoch UInt64,
	aggregated_rewards UInt64,
	aggregated_max_rewards UInt64,
	count_sync_committee UInt64,
	count_missing_source UInt64,
	count_missing_target UInt64,
	count_missing_head UInt64,
	count_expected_attestations UInt64,
	proposed_blocks_performance UInt64,
	missed_blocks_performance UInt64,
	number_active_vals UInt64)
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_epoch, f_pool_name);

CREATE TABLE IF NOT EXISTS t_proposer_duties(
	f_val_idx UInt64,
	f_proposer_slot UInt64,
	f_proposed BOOL)
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_proposer_slot, f_val_idx);


CREATE TABLE IF NOT EXISTS t_status(
	f_id UInt64 PRIMARY KEY,
	f_status TEXT)
	ENGINE = ReplacingMergeTree()
	ORDER BY f_id;

INSERT INTO t_status VALUES(0, 'in_activation_queue');
INSERT INTO t_status VALUES(1, 'active');
INSERT INTO t_status VALUES(2, 'slashed');
INSERT INTO t_status VALUES(3, 'exited');

CREATE TABLE IF NOT EXISTS t_transactions(
    f_tx_idx UInt64,
    f_tx_type UInt64,
    f_chain_id UInt64,
    f_data TEXT DEFAULT '',
    f_gas UInt64,
    f_gas_price UInt64,
    f_gas_tip_cap UInt64,
    f_gas_fee_cap UInt64,
    f_value Float,
    f_nonce UInt64,
    f_to TEXT DEFAULT '',
	f_from TEXT DEFAULT '',
	f_contract_address TEXT DEFAULT '',
    f_hash TEXT,
    f_size UInt64,
	f_slot UInt64,
	f_el_block_number UInt64,
	f_timestamp UInt64)
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_slot, f_el_block_number, f_hash);

CREATE TABLE IF NOT EXISTS t_validator_last_status(
	f_val_idx UInt64,
	f_epoch UInt64,
	f_balance_eth Float,
	f_status UInt8,
	f_slashed BOOL,
	f_activation_epoch UInt64,
	f_withdrawal_epoch UInt64,
	f_exit_epoch UInt64,
	f_public_key TEXT)
	ENGINE = MergeTree()
	ORDER BY (f_val_idx);

CREATE TABLE IF NOT EXISTS t_validator_rewards_summary(
	f_val_idx UInt64,
	f_epoch UInt64,
	f_balance_eth Float,
	f_reward Int64,
	f_max_reward UInt64,
	f_max_att_reward UInt64,
	f_max_sync_reward UInt64,
	f_att_slot UInt64,
	f_base_reward UInt64,
	f_in_sync_committee BOOL,
	f_missing_source BOOL,
	f_missing_target BOOL, 
	f_missing_head BOOL,
	f_status UInt8,
	f_block_api_reward UInt64,
	f_block_experimental_reward UInt64)
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_epoch, f_val_idx);

CREATE TABLE IF NOT EXISTS t_withdrawals(
	f_slot UInt64,
	f_index UInt64,
	f_val_idx UInt64,
	f_address TEXT,
	f_amount UInt64)
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_slot);

CREATE TABLE IF NOT EXISTS t_reorgs(
	f_slot UInt64 PRIMARY KEY,
	f_depth UInt64,
	f_old_head_block_root TEXT,
	f_new_head_block_root TEXT,
	f_old_head_state_root TEXT,
	f_new_head_state_root TEXT)
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_slot);

CREATE TABLE IF NOT EXISTS t_finalized_checkpoint(
	f_id UInt64,
	f_block_root TEXT,
	f_state_root TEXT,
	f_epoch UInt64)
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_epoch);

CREATE TABLE IF NOT EXISTS t_genesis(
	f_genesis_time UInt64 PRIMARY KEY)
	ENGINE = ReplacingMergeTree();



CREATE TABLE IF NOT EXISTS t_eth2_pubkeys(
	f_val_idx UInt64 PRIMARY KEY,
	f_public_key TEXT,
	f_pool_name TEXT,
	f_pool TEXT)
	ENGINE = ReplacingMergeTree();

CREATE TABLE IF NOT EXISTS t_head_events(
	f_slot UInt64,
    f_block TEXT,
    f_state TEXT,
    f_epoch_transition BOOLEAN,
    f_current_duty_dependent_root TEXT,
    f_previous_duty_dependent_root TEXT,
	f_arrival_timestamp UInt64)
	ENGINE = ReplacingMergeTree()
	ORDER BY (f_block);