CREATE TABLE t_consolidation_requests
(
    f_slot UInt64,
    f_source_address TEXT,
    f_source_pubkey TEXT,
    f_target_pubkey TEXT,
)
ENGINE = ReplacingMergeTree()
ORDER BY (f_slot, f_source_address, f_target_pubkey);