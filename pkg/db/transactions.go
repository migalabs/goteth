package db

import "github.com/pkg/errors"

var (
	CreateTransactionsTable = `
CREATE TABLE IF NOT EXISTS t_transactions(
    f_tx_idx SERIAL,
    f_tx_type VARCHAR(10),
    f_chain_id BIGINT,
    f_data TEXT,
    f_gas INT,
    f_gas_price BIGINT,
    f_gas_tip_cap BIGINT,
    f_gas_fee_cap BIGINT,
    f_value BIGINT,
    f_nonce INT,
    f_to TEXT,
    f_blob_gas INT,
    f_blob_gas_cap INT,
    f_blob_gas_fee_cap BIGINT,
    f_blob_hash TEXT,
    f_time TIMESTAMP,
    f_hash TEXT UNIQUE,
    f_size INT,
    f_from TEXT,
    CONSTRAINT t_transactions_pkey PRIMARY KEY (f_tx_idx, f_hash));`

	UpsertTransaction = `
INSERT INTO t_transactions(
	f_tx_type, f_chain_id, f_data, f_gas, f_gas_price, f_gas_tip_cap, f_gas_fee_cap, f_value, f_nonce, f_to, f_blob_gas, f_blob_gas_cap, f_blob_gas_fee_cap, f_blob_hash, f_time, f_hash, f_size, f_from
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
ON CONFLICT ON CONSTRAINT t_transactions_pkey DO NOTHING;`
)

/**
 *	Create Transactions Table
 */
func (p PostgresDBService) createTransactionsTable() error {
	// create tx table
	_, err := p.psqlPool.Exec(p.ctx, CreateTransactionsTable)
	if err != nil {
		return errors.Wrap(err, "error creating transactions table")
	}
	return nil
}
