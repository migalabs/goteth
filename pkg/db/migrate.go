package db

import (
	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Use as a reference because it contains sync.WaitGroup
func (s *DBService) makeMigrations() {

	m, err := migrate.New(
		"file://pkg/db/migrations",
		s.connectionUrl)
	if err != nil {
		log.Fatalf(err.Error())
	}
	log.Infof("applying database migrations...")
	if err := m.Up(); err != nil {
		if err != migrate.ErrNoChange {
			log.Fatalf(err.Error())
		}
	}
	connErr, dbErr := m.Close()

	if connErr != nil {
		log.Fatalf(connErr.Error())
	}
	if dbErr != nil {
		log.Fatalf(dbErr.Error())
	}
}
