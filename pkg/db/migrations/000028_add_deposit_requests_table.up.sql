CREATE TABLE t_deposit_requests
(
    f_slot UInt64,
    f_pubkey TEXT,
    f_withdrawal_credentials TEXT,
    f_amount UInt64,
    f_signature TEXT,
    f_index UInt64
)
ENGINE = ReplacingMergeTree()
ORDER BY (f_slot, f_index);