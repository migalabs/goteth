CREATE TABLE t_withdrawal_requests
(
    f_slot UInt64,
    f_index UInt64,
    f_source_address TEXT,
    f_validator_pubkey TEXT,
    f_amount UInt64,
    f_result UInt8
)
ENGINE = ReplacingMergeTree()
ORDER BY (f_slot, f_index);