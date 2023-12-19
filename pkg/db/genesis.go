package db

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
)

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

// Postgres intregration variables
var (
	genesisTable       = "t_genesis"
	insertGenesisQuery = `
	INSERT INTO %s (
		f_genesis_time)
		VALUES`

	selectGenesisQuery = `
	SELECT f_genesis_time
	FROM %s;
`
)

type InsertGenesis struct {
	genesis []int64
}

func (d InsertGenesis) Table() string {
	return genesisTable
}

func (d *InsertGenesis) Append(newGenesis int64) {
	d.genesis = append(d.genesis, newGenesis)
}

func (d InsertGenesis) Columns() int {
	return len(d.Input().Columns())
}

func (d InsertGenesis) Rows() int {
	return len(d.genesis)
}

func (d InsertGenesis) Query() string {
	return fmt.Sprintf(insertGenesisQuery, genesisTable)
}
func (d InsertGenesis) Input() proto.Input {
	// one object per column
	var (
		f_genesis_time proto.ColUInt64
	)

	for _, genesis := range d.genesis {
		f_genesis_time.Append(uint64(genesis))
	}

	return proto.Input{

		{Name: "f_genesis_time", Data: f_genesis_time},
	}
}

func (p *DBService) RetrieveGenesis() (int64, error) {

	var result int64
	query := fmt.Sprintf(selectGenesisQuery, genesisTable)
	var err error
	var dest []struct {
		F_genesis_time uint64 `ch:"f_genesis_time"`
	}
	startTime := time.Now()

	p.highMu.Lock()
	err = p.highLevelClient.Select(p.ctx, &dest, query)
	p.highMu.Unlock()

	if err == nil && len(dest) > 0 {
		log.Infof("retrieved %d rows in %f seconds, query: %s", len(dest), time.Since(startTime).Seconds(), query)
		result = int64(dest[0].F_genesis_time)
	}

	return result, err
}

func (p *DBService) InitGenesis(apiGenesis time.Time) {
	// Get genesis from the API

	dbGenesis, err := p.RetrieveGenesis()
	if err != nil {
		log.Fatalf("could not get genesis from database: %s", err)
	}

	if dbGenesis == 0 { // table is empty, probably first time use
		var genesis InsertGenesis
		genesis.Append(apiGenesis.Unix())
		p.Persist(genesis)
		dbGenesis = apiGenesis.Unix()
	}

	if apiGenesis.Unix() != dbGenesis {
		log.Panicf("the genesis time in the database does not match the API, is the beacon node in the correct network?")
	}
}
