package model

// Postgres intregration variables
var (
	CreateStatusTable = `
	CREATE TABLE IF NOT EXISTS t_status(
		f_id INT,
		f_status TEXT PRIMARY KEY);`

	UpsertStatus = `
	INSERT INTO t_status (
		f_id, 
		f_status)
		VALUES ($1, $2)
		ON CONFLICT ON CONSTRAINT t_status_pkey
		DO NOTHING
	`
)
