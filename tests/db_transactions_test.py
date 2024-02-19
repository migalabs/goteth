import unittest
import unit_db_test.testcase as dbtest

class CheckIntegrityOfDB(dbtest.DBintegrityTest):
    db_config_file = ".env"

    def test_transactions_per_block(self):
        """ make sure that the number of tracked transactions match the ones included in the corresponding block """
        sql_query = """
        with tx_count as (
			select f_slot, count(distinct(f_hash)) as tx_count
        	from t_transactions
            group by f_slot
		)
        
        select t_block_metrics.f_slot, f_el_transactions, tx_count
        from t_block_metrics
        inner join tx_count
        on t_block_metrics.f_slot = tx_count.f_slot
        where f_el_transactions != tx_count
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
        """The test ensures that there are no transactions from missed or 0 transactions blocks"""
        sql_query = """
    	select *
		from t_block_metrics
		where f_slot in (
			select f_slot
			from t_transactions
		) and (f_el_transactions = 0 or f_proposed = false)
		order by f_slot desc
        """
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertNoRows(df)

if __name__ == '__main__':
    unittest.main()


