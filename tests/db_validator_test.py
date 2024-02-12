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


if __name__ == '__main__':
    unittest.main()
