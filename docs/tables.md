# Block Metrics | Orphans (`t_block_metrics`, `t_orphans`)

Config: `engine = ReplacingMergeTree ORDER BY f_slot`

| Column Name                  | Type of Data | Description                                            |
| ---------------------------- | ------------ | ------------------------------------------------------ |
| f_timestamp                  | uint64       | unix time of the slot                                  |
| f_epoch                      | uint64       | epoch number                                           |
| f_slot                       | uint64       | slot number                                            |
| f_graffiti                   | string       | graffiti                                               |
| f_proposer_index             | uint64       | validator index of the proposer                        |
| f_proposed                   | bool         | whether the block was proposed or not                  |
| f_attestations               | uint64       | number of attestations included in the block           |
| f_deposits                   | uint64       | number of deposits included in the block               |
| f_consolidation_requests_num | uint64       | number of consolidation requests included in the block |
| f_withdrawal_requests_num    | uint64       | number of withdrawal requests included in the block    |
| f_deposit_requests_num       | uint64       | number of deposit requests included in the block       |
| f_proposer_slashings         | uint64       | number of proposer slashings included in the block     |
| f_attester_slashings         | uint64       | number of attester slashings included in the block     |
| f_voluntary_exits            | uint64       | number of voluntary exits included in the block        |
| f_sync_bits                  | uint64       | number of sync bits = 1 included in the block          |
| f_el_fee_recp                | string       | fee recipient                                          |
| f_el_gas_limit               | uint64       | gas limit                                              |
| f_el_gas_used                | uint64       | gas used                                               |
| f_el_base_fee_per_gas        | uint64       | base fee per gas                                       |
| f_el_block_hash              | string       | hash of the execution payload                          |
| f_el_transactions            | uint64       | amount of transactions included                        |
| f_el_block_number            | uint64       | block number                                           |
| f_payload_size_bytes         | uint64       | amount of bytes of the execution payload               |
| f_ssz_size_bytes             | float32      | block size in bytes when serialized with SSZ           |
| f_snappy_size_bytes          | float32      | block size in bytes when compressed with snappy        |
| f_compression_time_ms        | float32      | milliseconds taken to compress the block               |
| f_decompression_time_ms      | float32      | milliseconds taken to decompress the block             |

# Epoch Metrics (`t_epoch_metrics_summary`)

Config: `engine = ReplacingMergeTree ORDER BY f_epoch`

| Column Name                        | Type of Data | Description                                                                                                            |
| ---------------------------------- | ------------ | ---------------------------------------------------------------------------------------------------------------------- |
| f_epoch                            | uint64       | epoch number                                                                                                           |
| f_slot                             | uint64       | slot number                                                                                                            |
| f_num_att                          | uint64       | number of attestations included in blocks in the epoch                                                                 |
| f_num_att_vals                     | uint64       | number of validators that attested to slots in the epoch                                                               |
| f_num_vals                         | uint64       | number of validators in the epoch                                                                                      |
| f_num_compounding_vals             | uint64       | number of validators compounding their rewards in the epoch                                                            |
| f_total_balance_eth                | float        | amount of ETH balance taking into account all active validators                                                        |
| f_att_effective_balance_eth        | uint64       | amount of ETH effective balance taking into account all active validators that attested                                |
| f_source_att_effective_balance_eth | uint64       | amount of ETH effective balance taking into account all active validators that achieved the source flag when attesting |
| f_target_att_effective_balance_eth | uint64       | amount of ETH effective balance taking into account all active validators that achieved the target flag when attesting |
| f_head_att_effective_balance_eth   | uint64       | amount of ETH effective balance taking into account all active validators that achieved the head flag when attesting   |
| f_total_effective_balance_eth      | uint64       | amount of ETH effective balance taking into account all active validators                                              |
| f_missing_source                   | uint64       | amount of single validator attestations with a missed source flag in the epoch                                         |
| f_missing_target                   | uint64       | amount of single validator attestations with a missed target flag in the epoch                                         |
| f_missing_head                     | uint64       | amount of single validator attestations with a missed head flag in the epoch                                           |
| f_timestamp                        | uint64       | unix time of the epoch                                                                                                 |
| f_num_slashed_vals                 | uint64       | amount of validators slashed up to this epoch                                                                          |
| f_num_active_vals                  | uint64       | amount of validators active in this epoch                                                                              |
| f_num_exited_vals                  | uint64       | amount of validators exited up to this epoch                                                                           |
| f_num_in_activation_vals           | uint64       | amount of validators in the activation queue during this epoch                                                         |
| f_sync_committee_participation     | uint64       | amount of validators that participated in the sync committee during this epoch                                         |
| f_deposits_num                     | uint64       | amount of eth2 deposits included in the epoch                                                                          |
| f_total_deposits_amount            | uint64       | amount of eth deposited in the epoch                                                                                   |
| f_withdrawals_num                  | uint64       | amount of withdrawals included in the epoch                                                                            |
| f_total_withdrawals_amount         | uint64       | amount of eth withdrawn in the epoch                                                                                   |
| f_new_proposer_slashings           | uint64       | amount of new [valid](https://github.com/migalabs/goteth/pull/146) proposer slashings included in the epoch            |
| f_new_attester_slashings           | uint64       | amount of new [valid](https://github.com/migalabs/goteth/pull/146) attester slashings included in the epoch            |
| f_consolidation_requests_num       | uint64       | number of consolidation requests included in the epoch                                                                 |
| f_withdrawal_requests_num          | uint64       | number of withdrawal requests included in the epoch                                                                    |
| f_deposit_requests_num             | uint64       | number of deposit requests included in the epoch                                                                       |
| f_consolidations_processed_num     | uint64       | number of consolidations processed in the epoch                                                                        |
| f_consolidations_processed_amount  | uint64       | total amount of ETH consolidated in the epoch (Gwei)                                                                   |

# Pool Summaries (`t_pool_summary`)

Config: `engine = ReplacingMergeTree ORDER BY f_epoch, f_pool_name`

| Column Name                                  | Type of Data | Description                                                                   |
| -------------------------------------------- | ------------ | ----------------------------------------------------------------------------- |
| f_pool_name                                  | string       | name of the pool                                                              |
| f_epoch                                      | uint64       | epoch number                                                                  |
| aggregated_rewards                           | uint64       | sum of rewards of validators in the given pool                                |
| aggregated_max_rewards                       | uint64       | sum of maximum rewards of validators in the given pool                        |
| aggregated_effective_balance                 | uint64       | sum of effective balances of validators in the given pool (Gwei)              |
| count_sync_committee                         | uint64       | number of validators participating in the sync committee for the given pool   |
| count_sync_committee_participations_included | uint64       | number of sync committee participations included for the pool in the epoch    |
| count_missing_source                         | uint64       | amount of validator with a missed source flag for the given pool              |
| count_missing_target                         | uint64       | amount of validator with a missed target flag for the given pool              |
| count_missing_head                           | uint64       | amount of validator with a missed head flag for the given pool                |
| count_expected_attestations                  | uint64       | amount of attestations expected for the given pool (one per active validator) |
| count_attestations_included                  | uint64       | amount of attestations included for the given pool corresponding to the epoch |
| proposed_blocks_performance                  | uint64       | sum of proposed blocks by validators in the given pool                        |
| missed_blocks_performance                    | uint64       | sum of missed blocks by validators in the given pool                          |
| number_active_vals                           | uint64       | number of active validators in the given pool                                 |
| number_compounding_vals                      | uint64       | number of validators compounding their rewards in the given pool              |
| f_avg_inclusion_delay                        | float32      | average of inclusion delay of active validators in the given pool             |

# Proposer Duties (`t_proposer_duties`)

Config: `engine = ReplacingMergeTree ORDER BY f_proposer_slot, f_val_idx`

| Column Name     | Type of Data | Description                                     |     |     |
| --------------- | ------------ | ----------------------------------------------- | --- | --- |
| f_val_idx       | uint64       | validator index                                 |
| f_proposer_slot | uint64       | slot at which the validator had a proposer duty |
| f_proposed      | bool         | whether the block was proposed or not           |

# Transactions (`t_transactions`)

Config: `engine = ReplacingMergeTree ORDER BY f_slot, f_el_block_number, f_hash`

| Column Name        | Type of Data | Description                                                                                                             |     |     |
| ------------------ | ------------ | ----------------------------------------------------------------------------------------------------------------------- | --- | --- |
| f_tx_idx           | uint64       | transaction index                                                                                                       |
| f_tx_type          | uint64       | transaction type <br>LegacyTxType = 0x00 <br> AccessListTxType = 0x01<br> DynamicFeeTxType = 0x02<br> BlobTxType = 0x03 |
| f_chain_id         | uint64       | chain ID                                                                                                                |
| f_data             | uint64       | call data                                                                                                               |
| f_gas              | uint64       | gas used                                                                                                                |
| f_gas_price        | uint64       | gas price (Wei)                                                                                                         |
| f_gas_tip_cap      | uint64       | gasTipCap per gas of the transaction (Wei)                                                                              |
| f_gas_fee_cap      | uint64       | fee cap per gas of the transaction (Wei)                                                                                |
| f_value            | float32      | value of the transaction                                                                                                |
| f_nonce            | uint64       | nonce of the transaction                                                                                                |
| f_to               | uint64       | address TO                                                                                                              |
| f_hash             | uint64       | hash of the transaction                                                                                                 |
| f_size             | uint64       | true encoded storage size of the transaction                                                                            |
| f_slot             | uint64       | slot in which the transaction was included                                                                              |
| f_el_block_number  | uint64       | block number in which the transaction was included                                                                      |
| f_timestamp        | uint64       | unix time of the slot at which the transaction was included                                                             |
| f_from             | uint64       | address FROM                                                                                                            |
| f_contract_address | uint64       | address of the contract                                                                                                 |
| f_blob_gas_used    | uint64       | amount of gas used                                                                                                      |
| f_blob_gas_price   | uint64       | price per gas (Wei)                                                                                                     |
| f_blob_gas_limit   | uint64       | limit of gas to use                                                                                                     |
| f_blob_gas_fee_cap | uint64       | fee cap per gas (Wei)                                                                                                   |

# Status (`t_status`)

Config: `engine = ReplacingMergeTree ORDER BY f_id`

| Column Name | Type of Data | Description                                                                                          |     |     |
| ----------- | ------------ | ---------------------------------------------------------------------------------------------------- | --- | --- |
| f_id        | uint64       | id of the status                                                                                     |
| f_status    | string       | name of the status <br> 0, 'in_activation_queue' <br> 1, 'active' <br> 2, 'exited' <br> 3, 'slashed' |

# Validator Last Status (`t_validator_last_status`)

Config: `engine = MergeTree ORDER BY f_val_idx`

| Column Name              | Type of Data | Description                                         |
| ------------------------ | ------------ | --------------------------------------------------- |
| f_val_idx                | uint64       | validator index                                     |
| f_epoch                  | uint64       | epoch number                                        |
| f_balance_eth            | float32      | eth balance of the validator                        |
| f_effective_balance      | uint64       | effective balance of the validator (Gwei)           |
| f_status                 | uint8        | status (see status table)                           |
| f_slashed                | bool         | whether the validator has ever been slashed or not  |
| f_activation_epoch       | uint64       | epoch at which the validator was activated          |
| f_withdrawal_epoch       | uint64       | epoch at which the validator can withdraw funds     |
| f_exit_epoch             | uint64       | epoch at which the validator exited the network     |
| f_public_key             | string       | public key of the validator                         |
| f_withdrawal_prefix      | uint8        | withdrawal prefix of the validator's credentials    |
| f_withdrawal_credentials | text         | withdrawal credentials of the validator (see below) |

## Appendix: Reference for `f_withdrawal_prefix`

The `f_withdrawal_prefix` column indicates the type of withdrawal credentials associated with the validator. The possible values are:

- `0x00`: **BLS_WITHDRAWAL_PREFIX** - Indicates a BLS withdrawal prefix.
- `0x01`: **ETH1_ADDRESS_WITHDRAWAL_PREFIX** - Indicates an ETH1 address withdrawal prefix.
- `0x02`: **COMPOUNDING_WITHDRAWAL_PREFIX** - Indicates a compounding withdrawal prefix.

# Validator Rewards Summary (`t_validator_rewards_summary`)

Config: `engine = ReplacingMergeTree ORDER BY f_epoch, f_val_idx`

This table stores the data of the rewards obtained by validators in the network. It will only have rows for validators that are either:

- Active in the current epoch.
- Inactive but with duties still assigned (edge case of validators that are in the sync committee but not active).
- Slashed and not yet exited.

| Column Name                              | Type of Data | Description                                                                                                           |
| ---------------------------------------- | ------------ | --------------------------------------------------------------------------------------------------------------------- |
| f_val_idx                                | uint64       | validator index                                                                                                       |
| f_epoch                                  | uint64       | epoch number                                                                                                          |
| f_balance_eth                            | float        | eth balance at the end of the given epoch                                                                             |
| f_effective_balance                      | uint64       | effective balance of the validator at the end of the epoch (Gwei)                                                     |
| f_withdrawal_prefix                      | uint8        | withdrawal prefix of the validator's withdrawal credentials (see above)                                               |
| f_reward                                 | int64        | reward obtained from the previous epoch to the given epoch, can be negative (Gwei)                                    |
| f_max_reward                             | uint64       | maximum consensus reward that could have been obtained from the previous epoch to the given epoch (Gwei). Takes into forced missed flags (e.g. if missed head because of a missed slot, the max reward will take that into account)              |
| f_max_att_reward                         | uint64       | maximum attestation that could have been obtained from the previous epoch to the given epoch (Gwei)                   |
| f_max_sync_reward                        | uint64       | maximum sync committee that could have been obtained from the previous epoch to the given epoch (Gwei)                |
| f_att_slot                               | uint64       | slot the validator had to attest to (2 epochs before)                                                                 |
| f_base_reward                            | uint64       | base reward taken into account to calculate the rewards (Gwei)                                                        |
| f_in_sync_committee                      | bool         | whether the validator participated in the sync committee in the given epoch                                           |
| f_sync_committee_participations_included | uint8        | number of sync committee participations included for the validator in the given epoch                                 |
| f_attestation_included                   | bool         | whether the attestation was included in the chain (2 epochs before)                                                   |
| f_missing_source                         | bool         | whether the validator missed the source flag while attesting (takes into account the attestation to 2 epochs before)  |
| f_missing_target                         | bool         | whether the validator missed the target flag while attesting (takes into account the attestation to 2 epochs before)  |
| f_missing_head                           | bool         | whether the validator missed the head flag while attesting (takes into account the attestation to 2 epochs before)    |
| f_status                                 | uint8        | validator status 2 epochs before (see status table)                                                                   |
| f_block_api_reward                       | uint64       | consensus block reward obtained from the Beacon API (only if the validator was a proposer in the given epoch) (Gwei)  |
| f_block_experimental_reward              | uint64       | consensus block reward manually calculated by goteth (only if the validator was a proposer in the given epoch) (Gwei) |
| f_inclusion_delay                        | uint8        | amount of slots after the attested one at which the attestation was included                                          |

# Validator Rewards Aggregation (`t_validator_rewards_aggregation`)

Config: `engine = ReplacingMergeTree ORDER BY f_start_epoch, f_val_idx`

Table that stores the data from `t_validator_rewards_summary` but aggregated by validator on an epoch range.

| Column Name                              | Type of Data | Description                                                                                                                     |     |     |
| ---------------------------------------- | ------------ | ------------------------------------------------------------------------------------------------------------------------------- | --- | --- |
| f_val_idx                                | uint64       | validator index                                                                                                                 |
| f_start_epoch                            | uint64       | aggregation start epoch number                                                                                                  |
| f_end_epoch                              | uint64       | aggregation end epoch number (inclusive)                                                                                        |
| f_reward                                 | int64        | reward obtained from the previous epoch in the given epoch range, can be negative(Gwei)                                         |
| f_max_reward                             | uint64       | maximum consensus reward that could have been obtained from the previous epoch in the given epoch range (Gwei)                  |
| f_max_att_reward                         | uint64       | maximum attestation that could have been obtained from the previous epoch in the given epoch range (Gwei)                       |
| f_max_sync_reward                        | uint64       | maximum sync committee that could have been obtained from the previous epoch in the given epoch range (Gwei)                    |
| f_base_reward                            | uint64       | base reward taken into account to calculate the rewards (Gwei)                                                                  |
| f_in_sync_committee_count                | uint16       | number of times the validator participated in the sync commmittee in the given epoch range                                      |
| f_sync_committee_participations_included | uint16       | number of sync committee participations included for the validator in the given epoch range                                     |
| f_attestations_included                  | uint16       | number of times the attestation was included in the chain (takes into account the attestation to 2 epochs before)               |
| f_missing_source_count                   | uint16       | the amount of times the validator missed the source flag while attesing (takes into account the attestation to 2 epochs before) |
| f_missing_target_count                   | uint16       | the amount of times the validator missed the target flag while attesing (takes into account the attestation to 2 epochs before) |
| f_missing_head_count                     | uint16       | the amount of times the validator missed the head flag while attesing (takes into account the attestation to 2 epochs before)   |
| f_block_api_reward                       | uint64       | consensus block reward obtained from the Beacon API (only if the validator was a proposer in the given epoch) (Gwei)            |
| f_block_experimental_reward              | uint64       | consensus block reward manually calculated by goteth (only if the validator was a proposer in the given epoch) (Gwei)           |
| f_inclusion_delay_sum                    | uint32       | the sum of amount of slots after the attestations at which the attestations were included                                       |

# Withdrawals (`t_withdrawals`)

Config: `engine = ReplacingMergeTree ORDER BY f_index`

| Column Name | Type of Data | Description                                    |     |     |
| ----------- | ------------ | ---------------------------------------------- | --- | --- |
| f_slot      | uint64       | slot number                                    |
| f_index     | uint64       | index of the withdrawal inside the slot        |
| f_val_idx   | uint64       | validator index                                |
| f_address   | string       | address to which the withdrawal should be sent |
| f_amount    | uint64       | amount to be withdrawn (Gwei)                  |

# Reorgs (`t_reorgs`)

Config: `engine = ReplacingMergeTree ORDER BY f_slot`

| Column Name           | Type of Data | Description                            |     |     |
| --------------------- | ------------ | -------------------------------------- | --- | --- |
| f_slot                | uint64       | slot at which the reorg happened       |
| f_depth               | uint64       | number of blocks back the reorg covers |
| f_old_head_block_root | string       | root of the old head block             |
| f_new_head_block_root | string       | root of the new head block             |
| f_old_head_state_root | string       | root of the old head state             |
| f_new_head_state_root | string       | root of the new head state             |

# Finalized Checkpoint (`t_finalized_checkpoint`)

Config: `engine = ReplacingMergeTree ORDER BY f_epoch`

| Column Name  | Type of Data | Description                 |     |     |
| ------------ | ------------ | --------------------------- | --- | --- |
| f_id         | uint64       | incremental id              |
| f_block_root | string       | root of the finalzied block |
| f_state_root | string       | root of the finalized state |
| f_epoch      | uint64       | epoch finalized             |

# Eth2 Pubkeys (`t_eth2_pubkeys`)

Config: `engine = ReplacingMergeTree ORDER BY f_val_idx`

| Column Name  | Type of Data | Description                       |     |     |
| ------------ | ------------ | --------------------------------- | --- | --- |
| f_val_idx    | uint64       | validator index                   |
| f_public_key | string       | public key of the validator       |
| f_pool_name  | string       | pool the validator belongs to     |
| f_pool       | string       | extra name for sub categorization |

# Head Events (`t_head_events`)

Config: `engine = ReplacingMergeTree ORDER BY f_block`

| Column Name                    | Type of Data | Description                                                                                   |     |     |
| ------------------------------ | ------------ | --------------------------------------------------------------------------------------------- | --- | --- |
| f_slot                         | uint64       | slot number                                                                                   |
| f_block                        | string       | root of the head block                                                                        |
| f_state                        | string       | root of the head state                                                                        |
| f_epoch_transition             | bool         | whether the new head represents an epoch transition or not (true when beginning of new epoch) |
| f_current_duty_dependent_root  | string       |
| f_previous_duty_dependent_root | string       |
| f_arrival_timestamp            | uint64       | timestamp at which goteth received the head signal (unix miliseconds)                         |

# Blob Sidecars (`t_blob_sidecars`)

Config: `engine = ReplacingMergeTree ORDER BY f_slot, f_index`

| Column Name      | Type of Data | Description                                                |     |     |
| ---------------- | ------------ | ---------------------------------------------------------- | --- | --- |
| f_blob_hash      | string       | versioned blob has                                         |
| f_tx_hash        | string       | hash of the transaction referencing this blob in this slot |
| f_slot           | uint64       | slot number                                                |
| f_index          | uint8        | index of the blob                                          |
| f_kzg_commitment | string       | kzg commitment of the blob                                 |
| f_kzg_proof      | string       | kzg proof of the blob                                      |
| f_ending_0s      | uint64       | amount of consecutive 0s at the end of the blob bytes      |

# Blob Sidecars Events (`t_blob_sidecars_events`)

Config: `engine = ReplacingMergeTree ORDER BY f_arrival_timestamp_ms, f_blob_hash, f_slot`

| Column Name            | Type of Data | Description                                       |     |     |
| ---------------------- | ------------ | ------------------------------------------------- | --- | --- |
| f_arrival_timestamp_ms | uint64       | timestamp at which goteth received the blob event |
| f_blob_hash            | string       | hash of the blob                                  |
| f_slot                 | uint64       | slot at which the blob was sent                   |
| f_block_root           | string       | block root hash                                   |
| f_index                | uint8        | index of the blob                                 |
| f_kzg_commitment       | string       | kzg commitment of the blob                        |

# Block Rewards (`t_block_rewards`)

Config: `engine = ReplacingMergeTree ORDER BY f_slot`

| Column Name        | Type of Data | Description                                                                                                                       |     |     |
| ------------------ | ------------ | --------------------------------------------------------------------------------------------------------------------------------- | --- | --- |
| f_slot             | uint64       | Slot                                                                                                                              |
| f_burnt_fees       | uint64       | Fees burnt within the block (Wei)                                                                                                 |
| f_burnt_fees       | uint64       | Fees burnt within the block (Wei)                                                                                                 |
| f_cl_manual_reward | uint64       | Block reward manually calculated in the tool regarding Consensus Layer (Gwei)                                                     |
| f_cl_api_reward    | uint64       | Block reward gathered from the Beacon API regarding Consensus Layer (Gwei)                                                        |
| f_relays           | []string     | List of relays that were offering this block's payload                                                                            |
| f_builder_pubkey   | string       | The first of the builder pubkeys list that were submitting this block's payload (usually the same builder through several relays) |
| f_bid_commission   | uint64       | Bid submitted with the payload: what the validator receives as a reward                                                           |

# Slashings (`t_slashings`)

Table that stores the data of the slashings that happened in the network.

Config: `engine = ReplacingMergeTree ORDER BY f_slot, f_slashed_validator_index`

| Column Name                  | Type of Data | Description                                                                                                                                                                        |     |     |
| ---------------------------- | ------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --- | --- |
| f_slashed_validator_index    | uint64       | validator that was slashed                                                                                                                                                         |
| f_slashed_by_validator_index | uint64       | validator that slashed the other validator                                                                                                                                         |
| f_slashing_reason            | string       | reason for the slashing (ProposerSlashing, AttesterSlashing)                                                                                                                       |
| f_slot                       | uint64       | slot at which the slashing happened                                                                                                                                                |
| f_epoch                      | uint64       | epoch at which the slashing happened                                                                                                                                               |
| f_valid                      | bool         | whether the slashing was valid or not, mainly due to [double slashings not being valid](https://migalabs.io/blog/post/slashed-validators-discrepancies-in-popular-block-explorers) |

# BLS To Execution Changes (`t_bls_to_execution_changes`)

Table that stores the BLS to execution changes that happened in the network.

Config: `engine = ReplacingMergeTree ORDER BY f_slot, f_epoch, f_validator_index`

| Column Name            | Type of Data | Description                                   |     |     |
| ---------------------- | ------------ | --------------------------------------------- | --- | --- |
| f_slot                 | uint64       | slot at which the change happened             |
| f_epoch                | uint64       | epoch at which the change happened            |
| f_validator_index      | uint64       | validator index that had the change           |
| f_from_bls_pubkey      | string       | BLS public key corresponding to the validator |
| f_to_execution_address | string       | execution address after the change            |

# ETH2 Deposits (`t_deposits`)

Table that stores the data of the deposits on the beaconchain.

Config: `engine = ReplacingMergeTree ORDER BY f_slot, f_epoch_processed, f_index`

| Column Name              | Type of Data | Description                                                                                            |     |     |
| ------------------------ | ------------ | ------------------------------------------------------------------------------------------------------ | --- | --- |
| f_slot                   | uint64       | slot at which the deposit was included                                                                 |
| f_epoch_processed        | uint64       | epoch at which the deposit was processed (for deposits previous to Electra, it will be `f_slot // 32`) |
| f_public_key             | string       | public key of the validator deposited                                                                  |
| f_withdrawal_credentials | string       | withdrawal credentials of the validator deposited                                                      |
| f_amount                 | uint64       | amount of ETH deposited (Gwei)                                                                         |
| f_signature              | string       | signature of the deposit data                                                                          |
| f_index                  | uint64       | index of the deposit in the slot (or in the epoch for deposits after Electra)                          |

# ETH1 Deposits (`t_eth1_deposits`)

Will only be filled if the correct beacon contract address is provided in the `.env` file or from the CLI.

Config: `engine = ReplacingMergeTree ORDER BY f_block_number, f_deposit_index`

| Column Name              | Type of Data | Description                                    |     |     |
| ------------------------ | ------------ | ---------------------------------------------- | --- | --- |
| f_block_number           | uint64       | block number at which the deposit was included |
| f_block_hash             | string       | hash of the block                              |
| f_tx_hash                | string       | hash of the transaction                        |
| f_log_index              | uint64       | log index of the deposit                       |
| f_sender                 | string       | address of the sender                          |
| f_recipient              | string       | address of the recipient                       |
| f_gas_used               | uint64       | gas used for the transaction                   |
| f_gas_price              | uint64       | gas price for the transaction                  |
| f_deposit_index          | uint64       | index of the deposit                           |
| f_validator_pubkey       | string       | public key of the validator                    |
| f_withdrawal_credentials | string       | withdrawal credentials of the validator        |
| f_signature              | string       | signature of the deposit data                  |
| f_amount                 | uint64       | amount of ETH deposited (Gwei)                 |

# Consolidation Requests (`t_consolidation_requests`)

Table that stores the data of consolidation requests in the network.

Config: `engine = ReplacingMergeTree ORDER BY f_slot, f_index`

| Column Name      | Type of Data | Description                                                           |
| ---------------- | ------------ | --------------------------------------------------------------------- |
| f_slot           | uint64       | Slot at which the consolidation request was made                      |
| f_index          | uint64       | Index of the consolidation request within the slot                    |
| f_source_address | string       | Address of the source from which the consolidation request originated |
| f_source_pubkey  | string       | Public key of the source involved in the consolidation request        |
| f_target_pubkey  | string       | Public key of the target involved in the consolidation request        |
| f_result         | uint8        | Result of the consolidation request (see reference below)             |

## Reference for `f_result`

### General Results

- `0`: **Unknown** - Consolidation request result is unknown.
- `1`: **Success** - Consolidation request was successful.

### Global Errors

- `10`: **TotalBalanceTooLow** - Total balance is too low to process the request.
- `11`: **QueueFull** - The consolidation request queue is full.
- `12`: **RequestUsedAsExit** - The consolidation request was used as an exit.

### Source Validator Errors

- `20`: **SrcNotFound** - Source validator was not found.
- `21`: **SrcInvalidCredentials** - Source validator has invalid credentials.
- `22`: **SrcInvalidSender** - Source validator sender is invalid.
- `23`: **SrcNotActive** - Source validator is not active.
- `24`: **SrcNotOldEnough** - Source validator is not old enough.
- `25`: **SrcHasPendingWithdrawal** - Source validator has a pending withdrawal.
- `26`: **SrcExitAlreadyInitiated** - Source validator has already initiated an exit.

### Target Validator Errors

- `30`: **TgtNotFound** - Target validator was not found.
- `31`: **TgtInvalidCredentials** - Target validator has invalid credentials.
- `32`: **TgtInvalidSender** - Target validator sender is invalid.
- `33`: **TgtNotCompounding** - Target validator is not compounding.
- `34`: **TgtNotActive** - Target validator is not active.
- `35`: **TgtExitAlreadyInitiated** - Target validator has already initiated an exit.

# Deposit Requests (`t_deposit_requests`)

Table that stores the data of deposit requests in the network.

Config: `engine = ReplacingMergeTree ORDER BY f_slot, f_index`

| Column Name              | Type of Data | Description                                        |
| ------------------------ | ------------ | -------------------------------------------------- |
| f_slot                   | uint64       | Slot at which the deposit request was made         |
| f_pubkey                 | text         | Public key of the validator making the deposit     |
| f_withdrawal_credentials | text         | Withdrawal credentials associated with the deposit |
| f_amount                 | uint64       | Amount of ETH requested for deposit (Gwei)         |
| f_signature              | text         | Signature of the deposit data                      |
| f_index                  | uint64       | Index of the deposit request within the slot       |

# Withdrawal Requests (`t_withdrawal_requests`)

Table that stores the data of withdrawal requests in the network.

Config: `engine = ReplacingMergeTree ORDER BY f_slot, f_index`

| Column Name        | Type of Data | Description                                                        |
| ------------------ | ------------ | ------------------------------------------------------------------ |
| f_slot             | uint64       | Slot at which the withdrawal request was made                      |
| f_index            | uint64       | Index of the withdrawal request within the slot                    |
| f_source_address   | string       | Address of the source from which the withdrawal request originated |
| f_validator_pubkey | string       | Public key of the validator involved in the withdrawal request     |
| f_amount           | uint64       | Amount of ETH requested for withdrawal (Gwei)                      |
| f_result           | uint8        | Result of the withdrawal request (see reference below)             |

## Reference for `f_result`

### General Results

- `0`: **Unknown** - Withdrawal request result is unknown.
- `1`: **Success** - Withdrawal request was successful.

### Errors

- `2`: **QueueFull** - The withdrawal request queue is full.
- `3`: **ValidatorNotFound** - The validator associated with the request was not found.
- `4`: **InvalidCredentials** - The validator has invalid credentials.
- `5`: **ValidatorNotActive** - The validator is not active.
- `6`: **ExitAlreadyInitiated** - The validator has already initiated an exit.
- `7`: **ValidatorNotOldEnough** - The validator is not old enough.
- `8`: **PendingWithdrawalExists** - A pending withdrawal already exists for the validator.
- `9`: **InsufficientBalance** - The validator has insufficient balance for the withdrawal.
- `10`: **ValidatorNotCompounding** - The validator is not compounding.
- `11`: **NoExcessBalance** - The validator has no excess balance to withdraw.

# Consolidations Processed (`t_consolidations_processed`)

Table that stores the data of processed consolidations in the network.

Config: `engine = ReplacingMergeTree ORDER BY f_epoch, f_index`

| Column Name           | Type of Data | Description                                                 |
| --------------------- | ------------ | ----------------------------------------------------------- |
| f_epoch               | uint64       | Epoch at which the consolidation was processed              |
| f_index               | uint64       | Index of the consolidation within the epoch                 |
| f_source_index        | uint64       | Index of the source validator involved in the consolidation |
| f_target_index        | uint64       | Index of the target validator involved in the consolidation |
| f_consolidated_amount | uint64       | Amount of ETH consolidated (Gwei)                           |
| f_valid               | bool         | Whether the consolidation was valid (default is `true`)     |
