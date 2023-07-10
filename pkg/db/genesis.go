package db

import (
	"log"
)

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

// Postgres intregration variables
var (
	InsertGenesisQuery = `
	INSERT INTO t_genesis (
		f_genesis_time)
		VALUES ($1)
		ON CONFLICT ON CONSTRAINT t_genesis_pkey
		DO NOTHING;
	`

	GetGenesisQuery = `
	SELECT f_genesis_time
	FROM t_genesis;
`
)

// in case the table did not exist
func (p *PostgresDBService) ObtainGenesis() int64 {
	// create the tables
	rows, err := p.psqlPool.Query(p.ctx, GetGenesisQuery)
	if err != nil {
		rows.Close()
		log.Panicf("error obtaining genesis from database: %s", err)
	}
	genesis := int64(0)
	rows.Next()
	rows.Scan(&genesis)
	rows.Close()
	return genesis
}

// in case the table did not exist
func (p *PostgresDBService) InsertGenesis(genesisTime int64) {
	// create the tables
	rows, err := p.psqlPool.Query(p.ctx, InsertGenesisQuery, genesisTime)
	if err != nil {
		rows.Close()
		log.Panicf("error inserting genesis into database: %s", err)
	}
	rows.Close()
}
