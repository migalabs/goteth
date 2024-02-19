package db

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
)

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

func genesisInput(genesis []int64) proto.Input {
	// one object per column
	var (
		f_genesis_time proto.ColUInt64
	)

	for _, genesis := range genesis {
		f_genesis_time.Append(uint64(genesis))
	}

	return proto.Input{

		{Name: "f_genesis_time", Data: f_genesis_time},
	}
}

func (p *DBService) RetrieveGenesis() (int64, error) {

	var dest []struct {
		F_genesis_time uint64 `ch:"f_genesis_time"`
	}

	err := p.highSelect(
		fmt.Sprintf(selectGenesisQuery, genesisTable),
		&dest)

	if len(dest) > 0 {
		return int64(dest[0].F_genesis_time), err
	}
	return 0, err

}

func (p *DBService) InitGenesis(apiGenesis time.Time) error {
	// Get genesis from the API

	dbGenesis, err := p.RetrieveGenesis()
	if err != nil {
		log.Errorf("could not get genesis from database: %s", err)
		return err
	}

	if dbGenesis == 0 { // table is empty, probably first time use
		insertGenesis := PersistableObject[int64]{
			input: genesisInput,
			table: genesisTable,
			query: insertGenesisQuery,
		}
		insertGenesis.Append(apiGenesis.Unix())
		err := p.Persist(insertGenesis.ExportPersist())
		if err != nil {
			log.Errorf("could not persist genesis into the db: %s", err)
			return err
		}
		dbGenesis = apiGenesis.Unix()
	}

	if apiGenesis.Unix() != dbGenesis {
		log.Errorf("the genesis time in the database does not match the API, is the beacon node in the correct network?")
	}

	return err
}
