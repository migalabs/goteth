package db

import (
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

var (
	poolsTables = "t_pool_summary"

	insertPoolSummary = `
		INSERT INTO %s
			SELECT 
				t_eth2_pubkeys.f_pool_name, f_epoch,
				SUM(CASE WHEN (f_reward <= f_max_reward) THEN f_reward ELSE 0 END) as aggregated_rewards,
				SUM(CASE WHEN (f_reward <= f_max_reward) THEN f_max_reward ELSE 0 END) as aggregated_max_rewards,
				COUNT(CASE WHEN f_in_sync_committee = TRUE THEN 1 ELSE null END) as count_sync_committee,
				COUNT(CASE WHEN f_missing_source = TRUE THEN 1 ELSE null END) as count_missing_source,
				COUNT(CASE WHEN f_missing_target = TRUE THEN 1 ELSE null END) as count_missing_target,
				COUNT(CASE WHEN f_missing_head = TRUE THEN 1 ELSE null END) as count_missing_head,
				COUNT(*) as count_expected_attestations,
				SUM(CASE WHEN f_attestation_included = TRUE THEN 1 ELSE 0 END) as count_included_attestations,
				SUM(CASE WHEN t_proposer_duties.f_proposed = TRUE THEN 1 ELSE 0 END) as proposed_blocks_performance,
				SUM(CASE WHEN t_proposer_duties.f_proposed = FALSE and t_validator_rewards_summary.f_val_idx = t_proposer_duties.f_val_idx THEN 1 ELSE 0 END) as missed_blocks_performance,
				count(distinct(t_validator_rewards_summary.f_val_idx)) as number_active_vals,
				AVG(f_inclusion_delay) as avg_inclusion_delay
			FROM t_validator_rewards_summary
			LEFT JOIN t_eth2_pubkeys 
				ON t_validator_rewards_summary.f_val_idx = t_eth2_pubkeys.f_val_idx
			LEFT JOIN t_proposer_duties 
				ON t_validator_rewards_summary.f_val_idx = t_proposer_duties.f_val_idx 
				AND t_validator_rewards_summary.f_epoch = toUInt64(t_proposer_duties.f_proposer_slot/32)
			WHERE f_epoch = $1 AND f_status = 1 AND f_pool_name != ''
			GROUP BY t_eth2_pubkeys.f_pool_name, f_epoch`
)

func (p *DBService) InsertPoolSummary(epoch phase0.Epoch) error {

	query := fmt.Sprintf(insertPoolSummary, poolsTables)
	var err error
	startTime := time.Now()

	p.highMu.Lock()
	err = p.highLevelClient.Exec(p.ctx, query, epoch)
	p.highMu.Unlock()

	if err == nil {
		log.Infof("pool summaries created for epoch %d, %f seconds", epoch, time.Since(startTime).Seconds())

	}

	return err
}
