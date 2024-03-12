import unittest
import unit_db_test.testcase as dbtest

class CheckIntegrityOfDB(dbtest.DBintegrityTest):
    db_config_file = ".env"

    def test_no_blobs_with_tx_type_3(self):
        """ Test that there are no blobs with tx type != 3 (blob type) """
        sql_query = """
        select t_transactions.f_hash
		from t_transactions
		inner join t_blob_sidecars
		on t_transactions.f_hash == t_blob_sidecars.f_tx_hash
        where f_tx_type != 3
        """
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertNoRows(df)
        
    def test_no_more_than_6_blobs_per_block(self):
        """ Test that there are no more than 6 blobs per slot """
        sql_query = """
			select t_transactions.f_slot, count() as number_blobs
			from t_transactions
			inner join t_blob_sidecars
			on t_transactions.f_hash == t_blob_sidecars.f_tx_hash
			group by t_transactions.f_slot
			having number_blobs > 6
			order by t_transactions.f_slot DESC
		"""
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertNoRows(df)

    
        
if __name__ == '__main__':
    unittest.main()
