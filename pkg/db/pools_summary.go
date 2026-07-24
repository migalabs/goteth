package db

import (
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

const (
	// UnknownPoolName groups validators that have no entry (or an empty pool
	// name) in t_eth2_pubkeys at aggregation time. Keeping them in a synthetic
	// pool instead of dropping them guarantees that the sum across pools
	// always matches the network totals in t_epoch_metrics_summary.
	UnknownPoolName = "unknown"

	// poolSummaryRefreshBatchEpochs is the number of epochs recomputed per
	// INSERT when refreshing a range. Bounds the memory used by the
	// aggregation while keeping the refresh fast (a batch takes seconds).
	poolSummaryRefreshBatchEpochs = 250
)

var (
	poolsTables = "t_pool_summary"

	// The missed-blocks condition re-checks that the joined proposer duty row
	// really belongs to this validator and epoch: ClickHouse fills the right
	// side of a non-matched LEFT JOIN with type defaults (f_val_idx = 0,
	// f_proposed = false), which validator index 0 satisfies spuriously and
	// gets a fake missed block on every epoch it does not propose.
	poolSummarySelect = `
			SELECT
				if(t_eth2_pubkeys.f_pool_name = '', '%s', t_eth2_pubkeys.f_pool_name) as f_pool_name, f_epoch,
				SUM(CASE WHEN (f_reward <= f_max_reward) THEN f_reward ELSE 0 END) as aggregated_rewards,
				SUM(CASE WHEN (f_reward <= f_max_reward) THEN f_max_reward ELSE 0 END) as aggregated_max_rewards,
				SUM(f_effective_balance) as aggregated_effective_balance,
				COUNT(CASE WHEN f_in_sync_committee = TRUE THEN 1 ELSE null END) as count_sync_committee,
				SUM(f_sync_committee_participations_included) as count_sync_committee_participations_included,
				COUNT(CASE WHEN f_missing_source = TRUE THEN 1 ELSE null END) as count_missing_source,
				COUNT(CASE WHEN f_missing_target = TRUE THEN 1 ELSE null END) as count_missing_target,
				COUNT(CASE WHEN f_missing_head = TRUE THEN 1 ELSE null END) as count_missing_head,
				COUNT(*) as count_expected_attestations,
				SUM(CASE WHEN f_attestation_included = TRUE THEN 1 ELSE 0 END) as count_included_attestations,
				SUM(CASE WHEN t_proposer_duties.f_proposed = TRUE THEN 1 ELSE 0 END) as proposed_blocks_performance,
				SUM(CASE WHEN t_proposer_duties.f_proposed = FALSE
					AND t_validator_rewards_summary.f_val_idx = t_proposer_duties.f_val_idx
					AND t_validator_rewards_summary.f_epoch = toUInt64(t_proposer_duties.f_proposer_slot/32) THEN 1 ELSE 0 END) as missed_blocks_performance,
				count(distinct(t_validator_rewards_summary.f_val_idx)) as number_active_vals,
				count(CASE WHEN f_withdrawal_prefix = 2 THEN 1 ELSE null END) as number_compounding_vals,
				AVG(f_inclusion_delay) as avg_inclusion_delay
			FROM t_validator_rewards_summary final
			LEFT JOIN t_eth2_pubkeys final
				ON t_validator_rewards_summary.f_val_idx = t_eth2_pubkeys.f_val_idx
			LEFT JOIN t_proposer_duties final
				ON t_validator_rewards_summary.f_val_idx = t_proposer_duties.f_val_idx
				AND t_validator_rewards_summary.f_epoch = toUInt64(t_proposer_duties.f_proposer_slot/32)
			WHERE %s AND f_status = 1
			GROUP BY if(t_eth2_pubkeys.f_pool_name = '', '%s', t_eth2_pubkeys.f_pool_name), f_epoch`

	deletePoolSummaryRangeQuery = `
		DELETE FROM %s
		WHERE f_epoch >= $1 AND f_epoch <= $2;
	`

	// Order-independent fingerprint of the validator->pool mapping, used to
	// detect pool re-tagging so recent pool summaries can be refreshed.
	selectPoolsFingerprintQuery = `
		SELECT concat(toString(count()), '-', toString(sum(cityHash64(f_val_idx, f_pool_name)))) as f_fingerprint
		FROM t_eth2_pubkeys FINAL`
)

func insertPoolSummaryQuery(whereClause string) string {
	selectStmt := fmt.Sprintf(poolSummarySelect, UnknownPoolName, whereClause, UnknownPoolName)
	return fmt.Sprintf("INSERT INTO %s%s", poolsTables, selectStmt)
}

func (p *DBService) InsertPoolSummary(epoch phase0.Epoch) error {

	query := insertPoolSummaryQuery("f_epoch = $1")
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

func (p *DBService) insertPoolSummaryRange(fromEpoch phase0.Epoch, toEpoch phase0.Epoch) error {

	query := insertPoolSummaryQuery("f_epoch >= $1 AND f_epoch <= $2")
	var err error

	p.highMu.Lock()
	err = p.highLevelClient.Exec(p.ctx, query, fromEpoch, toEpoch)
	p.highMu.Unlock()

	return err
}

func (p *DBService) DeletePoolSummaryRange(fromEpoch phase0.Epoch, toEpoch phase0.Epoch) error {

	deleteObj := DeletableObject{
		query: deletePoolSummaryRangeQuery,
		table: poolsTables,
		args:  []any{fromEpoch, toEpoch},
	}

	err := p.Delete(deleteObj)
	if err != nil {
		log.Errorf("error deleting pool summaries: %s", err.Error())
	}

	return err
}

// RefreshPoolSummaryRange recomputes t_pool_summary for [fromEpoch, toEpoch]
// from the current contents of t_validator_rewards_summary and
// t_eth2_pubkeys. Existing rows in the range are deleted first: the table is
// a ReplacingMergeTree keyed on (f_epoch, f_pool_name), so re-inserting after
// a pool was renamed or split would leave ghost rows under the old pool name
// and double count its validators.
func (p *DBService) RefreshPoolSummaryRange(fromEpoch phase0.Epoch, toEpoch phase0.Epoch) error {

	if toEpoch < fromEpoch {
		return fmt.Errorf("invalid pool summary refresh range: %d-%d", fromEpoch, toEpoch)
	}

	startTime := time.Now()

	err := p.DeletePoolSummaryRange(fromEpoch, toEpoch)
	if err != nil {
		return err
	}

	for batchStart := fromEpoch; batchStart <= toEpoch; batchStart += poolSummaryRefreshBatchEpochs {
		batchEnd := batchStart + poolSummaryRefreshBatchEpochs - 1
		if batchEnd > toEpoch {
			batchEnd = toEpoch
		}
		err = p.insertPoolSummaryRange(batchStart, batchEnd)
		if err != nil {
			return fmt.Errorf("refreshing pool summaries %d-%d: %w", batchStart, batchEnd, err)
		}
		log.Debugf("pool summaries refreshed for epochs %d-%d", batchStart, batchEnd)
	}

	log.Infof("pool summaries refreshed for epochs %d-%d, %f seconds",
		fromEpoch, toEpoch, time.Since(startTime).Seconds())

	return nil
}

// RetrievePoolsFingerprint returns a cheap fingerprint of the current
// validator->pool mapping. It changes whenever t_eth2_pubkeys gains rows or
// any validator is re-tagged, and is used to trigger pool summary refreshes.
func (p *DBService) RetrievePoolsFingerprint() (string, error) {

	var dest []struct {
		F_fingerprint string `ch:"f_fingerprint"`
	}

	err := p.highSelect(selectPoolsFingerprintQuery, &dest)

	if len(dest) > 0 {
		return dest[0].F_fingerprint, err
	}
	return "", err
}
