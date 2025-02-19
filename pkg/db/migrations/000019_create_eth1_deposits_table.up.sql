CREATE TABLE t_eth1_deposits
(
    f_block_number      UInt64,
    f_block_hash        TEXT,
    f_tx_hash           TEXT,
    f_log_index         UInt64,
    f_sender            TEXT,
    f_recipient         TEXT,
    f_gas_used          UInt64,
    f_gas_price         UInt64,
    f_deposit_index          UInt64,
    f_validator_pubkey       TEXT,
    f_withdrawal_credentials TEXT,
    f_signature             TEXT,
    f_amount                UInt64
)
ENGINE = ReplacingMergeTree()
ORDER BY (f_block_number, f_deposit_index);