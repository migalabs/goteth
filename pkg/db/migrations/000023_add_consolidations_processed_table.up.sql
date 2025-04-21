CREATE TABLE t_consolidations_processed
(
    f_epoch UInt64,
    f_index UInt64,
    f_source_index UInt64,
    f_target_index UInt64,
    f_consolidated_amount UInt64,
    f_valid Bool DEFAULT true
)
ENGINE = ReplacingMergeTree()
ORDER BY (f_epoch, f_index);