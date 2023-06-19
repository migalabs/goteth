package db

import (
	migrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Use as a reference because it contains sync.WaitGroup
func (s *PostgresDBService) makeMigrations() {

	m, err := migrate.New(
		"file://pkg/db/migrations",
		s.connectionUrl)
	if err != nil {
		wlog.Fatalf(err.Error())
	}
	wlog.Infof("applying database migrations...")
	if err := m.Up(); err != nil {
		if err.Error() != "no change" {
			wlog.Fatalf(err.Error())
		}
	}
	connErr, dbErr := m.Close()

	if connErr != nil {
		wlog.Fatalf(connErr.Error())
	}
	if dbErr != nil {
		wlog.Fatalf(dbErr.Error())
	}

}
