import unittest
import unit_db_test.testcase as dbtest

class CheckIntegrityOfDB(dbtest.DBintegrityTest):
    db_config_file = ".env"

    def test_transactions_per_block(self):
        """ make sure that the number of tracked transactions match the ones included in the corresponding block """
        sql_query = """
        select t_block_metrics.f_slot, count(distinct(f_hash))
        from t_block_metrics
        inner join t_transactions
        on t_block_metrics.f_slot = t_transactions.f_slot
        group by t_block_metrics.f_slot
        having f_el_transactions != count(distinct(f_hash))
        """
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertNoRows(df)

    def test_missing_transactions_from_existing_blocks(self):
        """ Check if there are no blocks that aren't present in the transaction table, but had transactions and was proposed """
        sql_query = """
       select *
       from t_block_metrics
       where f_slot not in (
           select distinct(f_slot)
           from t_transactions
       ) and f_el_transactions > 0 and f_proposed = true
       order by f_slot desc
       """
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertNoRows(df)

    def test_number_of_blocks_across_tables(self):
        """The test ensures that the are the same number of blocks in across the existing tables"""
        sql_query = """
        select *
        from t_transactions
        inner join t_block_metrics
        on t_transactions.f_slot = t_block_metrics.f_slot
        where f_el_transactions = 0 or f_proposed = false
        """
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertNoRows(df)

if __name__ == '__main__':
    unittest.main()


