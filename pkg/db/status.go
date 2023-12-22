package db

var (
	UpsertStatus = `
	INSERT INTO t_status (
		f_id, 
		f_status)
		VALUES`
)
