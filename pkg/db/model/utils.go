package model

var (
	INSERT_OP = "INSERT"
	UPDATE_OP = "UPDATE"
	DROP_OP   = "DROP"
)

type Model interface { // simply to enforce a Model interface
	// For now we simply support insert operations
	InsertOp() bool // whether insert is activated for this model
}
