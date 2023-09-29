import unittest
import unit_db_test.testcase as dbtest

class CheckIntegrityOfDB(dbtest.DBintegrityTest):
    db_config_file = ".env"

    def test_orphan_blocks_in_block_metrics(self):
        """ Edgy case where two blocks are assigned to the same slot (one of them is an orphan) (after Merge) """
        sql_query = """
        select f_el_block_number, count(*)
        from t_block_metrics
        where f_proposed = true and f_slot > 4700012
        group by f_el_block_number
        having count(*) > 1
        """
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertNoRows(df)

    def test_missed_blocks_tagged_as_orphan(self):
        """ Edgy case where a missed blocks is added to the orphan table """
        sql_query = """
        select *
        from t_orphans
        where f_proposed = false
        """
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertNoRows(df)

    def test_block_gaps(self):
        """ make sure that there are no gaps between the indexed blocks """
        sql_query = """
        WITH Gaps AS (
            SELECT
                f_el_block_number AS preceding_block,
                LEAD(f_el_block_number) OVER (ORDER BY f_el_block_number) - 1 AS end_of_missing_range
            FROM
                t_block_metrics
            WHERE f_el_block_number > 16307594
        )
        SELECT
            preceding_block + 1 AS start_of_missing_range,
            end_of_missing_range
        FROM
            Gaps
        WHERE
            end_of_missing_range - preceding_block > 1
        ORDER BY
            preceding_block;        
        """
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertNoRows(df)

    def test_integrity_of_block_count(self):
        """Check if the total number of blocks present in the db correspond with the (proposed_with_tx+proposed_without_tx+not_proposed)"""
        sql_query = """
        select
            SUM(CASE WHEN f_proposed = true and f_el_transactions > 0 THEN 1 ELSE 0 END) as sum_block_with_tx,
            SUM(CASE WHEN f_proposed = true and f_el_transactions = 0 THEN 1 ELSE 0 END) as sum_block_no_tx,
            SUM(CASE WHEN f_proposed = false THEN 1 ELSE 0 END) as sum_missed,
            count(*) as blocks_in_range
        from t_block_metrics
        """
        df = self.db.get_df_from_sql_query(sql_query)
        df['total_present'] = df['sum_block_with_tx'] + df['sum_block_no_tx'] + df['sum_missed']
        self.assertEqual(df['total_present'].to_numpy(), df['blocks_in_range'].to_numpy())

if __name__ == '__main__':
    unittest.main()


