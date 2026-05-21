import unittest
import unit_db_test.testcase as dbtest

class CheckIntegrityOfDB(dbtest.DBintegrityTest):
    db_config_file = ".env"

    def test_reward_not_gt_than_max(self):
        """ After the last update to v2.0.0 the rewards must never exceed the f_max_reward """
        sql_query = """
        select *
        from t_validator_rewards_summary
        where f_status = 1 and abs(f_reward) > abs(f_max_reward)
        """
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertNoRows(df)

    def test_unpersisted_queries_for_validator_duties(self):
        """ There shouldn't be less proposer duties than the 32 slots per epoch, if so, there were some DB queries lost on the way """
        sql_query = """
        select trunc(f_proposer_slot/32) as epoch, count(*)
        from t_proposer_duties
        group by trunc(f_proposer_slot/32)
        having count(*) < 32
        """
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertNoRows(df)

    def test_weird_error_tracking_validator_duties(self):
        """ We could expect less that 32 duties/epoch because some queries were lost, but if we have more, there is something weird going on """
        sql_query = """
        select trunc(f_proposer_slot/32) as epoch, count(*)
        from t_proposer_duties
        group by trunc(f_proposer_slot/32)
        having count(*) > 32
        """
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertNoRows(df)
        
    def test_experimental_block_reward(self):
        """ We are manually calculating the block_rewards and we want to make sure it matches the block_api_reward"""
        sql_query = """
        select *
		from t_validator_rewards_summary
		where f_block_api_reward > 0 and f_block_api_reward != f_block_experimental_reward
        """
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertNoRows(df)

    def test_inclusion_delay_congruent_with_flags(self):
        """ The inclusion delay must be congruent with the flags """
        sql_query = """
        select *
        from t_validator_rewards_summary
        where (f_inclusion_delay > 5 and f_missing_source = false)
      	or (f_inclusion_delay > 1 and f_missing_head = false)
		or (f_inclusion_delay > 64 and f_missing_target = false)
        """
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertNoRows(df)

    # FAR_FUTURE_EPOCH = 2^64 - 1; sentinel meaning "not yet / possibly never".
    # Used as the default for f_activation_eligibility_epoch and f_activation_epoch
    # when those state transitions have not yet occurred.
    FAR_FUTURE_EPOCH = 18446744073709551615

    def test_active_validators_have_eligibility_set(self):
        """ Any validator that is active, exiting, exited or slashed must have a
        real activation_eligibility_epoch (not FAR_FUTURE_EPOCH). The spec
        forbids activation without prior eligibility, so seeing FAR_FUTURE here
        indicates the analyzer did not populate the new column for that row. """
        sql_query = f"""
        select f_val_idx, f_status, f_activation_epoch, f_activation_eligibility_epoch
        from t_validator_last_status
        where f_status in (1, 2, 3)
          and f_activation_eligibility_epoch = {self.FAR_FUTURE_EPOCH}
        """
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertNoRows(df)

    def test_eligibility_precedes_activation(self):
        """ For any validator that has an activation_epoch set, its
        activation_eligibility_epoch must also be set AND be less-than-or-equal
        to the activation_epoch. process_registry_updates requires eligibility
        before scheduling activation, so the inverse cannot legitimately occur. """
        sql_query = f"""
        select f_val_idx, f_activation_epoch, f_activation_eligibility_epoch
        from t_validator_last_status
        where f_activation_epoch != {self.FAR_FUTURE_EPOCH}
          and (f_activation_eligibility_epoch = {self.FAR_FUTURE_EPOCH}
               or f_activation_eligibility_epoch > f_activation_epoch)
        """
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertNoRows(df)

    def test_pending_split_is_observable(self):
        """ Sanity check that the new column is being populated, not just
        defaulted. After indexing a recent finalized range on a live network,
        we expect the queue (f_status = 0) to contain at least one row with
        a real eligibility epoch (pending_queued) — otherwise the analyzer is
        either not reading the field, or the migration default is hiding it.

        On a quiet testnet with no queued validators this can be empty; in
        that case rely on the synthetic SQL evidence in the PR proof of
        success instead of this test. """
        sql_query = f"""
        select count(*) as queued_count
        from t_validator_last_status
        where f_status = 0
          and f_activation_eligibility_epoch != {self.FAR_FUTURE_EPOCH}
          and f_activation_epoch = {self.FAR_FUTURE_EPOCH}
        """
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertGreater(df['queued_count'].iloc[0], 0)

if __name__ == '__main__':
    unittest.main()
