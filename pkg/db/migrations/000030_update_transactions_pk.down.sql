ALTER TABLE t_transactions DROP CONSTRAINT t_transactions_pkey;
ALTER TABLE t_transactions ADD CONSTRAINT t_transactions_pkey PRIMARY KEY (f_hash);