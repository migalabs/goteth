package db

import (
	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func (s *DBService) makeMigrations() error {

	m, err := migrate.New(
		"file://pkg/db/migrations",
		s.connectionUrl)
	if err != nil {
		log.Errorf(err.Error())
		return err
	}
	log.Infof("applying database migrations...")
	if err := m.Up(); err != nil {
		if err != migrate.ErrNoChange {
			log.Errorf(err.Error())
			return err
		}
	}
	connErr, dbErr := m.Close()

	if connErr != nil {
		log.Errorf(connErr.Error())
		return connErr
	}
	if dbErr != nil {
		log.Errorf(dbErr.Error())
		return dbErr
	}
	return err
}
