import unittest
import unit_db_test.testcase as dbtest

class CheckIntegrityOfDB(dbtest.DBintegrityTest):
    db_config_file = ".env"

    def test_arrival_timestamp_not_gt_block_timestamp(self):
        """ On the edge case where the arrival timestamp is not greater than the block timestamp, which is not possible """
        sql_query = """
        SELECT *
        FROM t_head_events
        JOIN t_block_metrics ON t_head_events.f_slot = t_block_metrics.f_slot
        WHERE t_head_events.f_arrival_timestamp / 1000 < t_block_metrics.f_timestamp;

        """
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertNoRows(df)

if __name__ == '__main__':
    unittest.main()


