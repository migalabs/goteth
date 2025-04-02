ALTER TABLE t_validator_rewards_summary DROP COLUMN f_attestation_included;


ALTER TABLE t_validator_rewards_aggregation DROP COLUMN f_attestations_included;

ALTER TABLE t_pool_summary DROP COLUMN count_attestations_included;