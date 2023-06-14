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

// In case we encounter a problem we need to be able to force the version.
// Force removes the dirty and allows to continue applying migrations.
func (s *PostgresDBService) ForceMigration(version int) {
	m, err := migrate.New(
		"file://pkg/db/migrations",
		s.connectionUrl)
	if err != nil {
		wlog.Fatalf(err.Error())
	}
	wlog.Infof("forcing database version %d...", version)
	if err := m.Force(version); err != nil {
		wlog.Fatalf(err.Error())
	}
	connErr, dbErr := m.Close()

	if connErr != nil {
		wlog.Fatalf(connErr.Error())
	}
	if dbErr != nil {
		wlog.Fatalf(dbErr.Error())
	}
}
