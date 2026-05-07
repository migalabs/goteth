ALTER TABLE t_block_rewards MODIFY COLUMN f_reward_fees UInt64 DEFAULT 0;
ALTER TABLE t_block_rewards MODIFY COLUMN f_burnt_fees UInt64 DEFAULT 0;
ALTER TABLE t_block_rewards MODIFY COLUMN f_bid_commission UInt64 DEFAULT 0;
